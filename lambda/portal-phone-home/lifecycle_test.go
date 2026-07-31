package main

import (
	"context"
	"testing"
	"time"
)

// Fixed clock — ApplyProbes takes `now` as a parameter precisely so the N-day
// transitions are testable without sleeping.
var t0 = time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

func rfc(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// find returns the changed row for an account, or nil if ApplyProbes decided
// nothing about it.
func find(changed []Account, id string) *Account {
	for i := range changed {
		if changed[i].AccountID == id {
			return &changed[i]
		}
	}
	return nil
}

// THE central safety property (spawn#457 trap 1): when every probe fails, decide
// nothing. Recreating the reaper's CloudFormation stack changes its role ARN
// suffix and breaks every customer's trust policy at once — a state machine that
// acted on assume-role failure alone would forget the entire customer base
// because we redeployed.
func TestApplyProbes_AllUnreachableChangesNothing(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	existing := []Account{
		{AccountID: "111111111111", Status: StatusActive},
		{AccountID: "222222222222", Status: StatusActive},
		{AccountID: "333333333333", Status: StatusActive},
	}
	probes := []ProbeResult{
		{AccountID: "111111111111", Reachable: false},
		{AccountID: "222222222222", Reachable: false},
		{AccountID: "333333333333", Reachable: false},
	}
	// Even sustained across many runs — K is irrelevant when the evidence is
	// explainable by our own breakage.
	for run := 0; run < 20; run++ {
		if changed := ApplyProbes(existing, probes, t0.Add(time.Duration(run)*10*time.Minute), pol); len(changed) != 0 {
			t.Fatalf("run %d: total-failure round changed %d row(s); must change none: %+v", run, len(changed), changed)
		}
	}
}

// The single-account deployment is the sharp edge of that guard: with one probe,
// "all probes failed" and "this one account failed" are the same observation, and
// the guard must resolve it as uninformative.
func TestApplyProbes_SingleAccountFailureIsUninformative(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	existing := []Account{{AccountID: "111111111111", Status: StatusActive}}
	probes := []ProbeResult{{AccountID: "111111111111", Reachable: false}}
	if changed := ApplyProbes(existing, probes, t0, pol); len(changed) != 0 {
		t.Errorf("sole-account failure changed %d row(s); cannot distinguish customer uninstall from our own breakage: %+v", len(changed), changed)
	}
}

// A failure alongside at least one success IS about that account — but only after
// K consecutive runs. Below K it accrues a counter and nothing else.
func TestApplyProbes_UnreachableRequiresKConsecutive(t *testing.T) {
	pol := DefaultLifecyclePolicy() // K=6
	const dead = "111111111111"
	const alive = "222222222222"

	acct := Account{AccountID: dead, Status: StatusActive}
	for run := 1; run <= pol.FailuresBeforeUnreachable; run++ {
		probes := []ProbeResult{
			{AccountID: dead, Reachable: false},
			{AccountID: alive, Reachable: true, LiveInstances: 1},
		}
		changed := ApplyProbes([]Account{acct, {AccountID: alive, Status: StatusActive}},
			probes, t0.Add(time.Duration(run)*10*time.Minute), pol)
		got := find(changed, dead)
		if got == nil {
			t.Fatalf("run %d: expected a change row for the failing account", run)
		}
		if got.ConsecutiveFailures != run {
			t.Errorf("run %d: consecutiveFailures = %d, want %d", run, got.ConsecutiveFailures, run)
		}
		if run < pol.FailuresBeforeUnreachable {
			if got.AccountStatus() != StatusActive {
				t.Errorf("run %d: status = %q, want still active below K=%d", run, got.AccountStatus(), pol.FailuresBeforeUnreachable)
			}
			if got.StatusChangedAt != "" {
				t.Errorf("run %d: StatusChangedAt set without a transition", run)
			}
		} else {
			if got.AccountStatus() != StatusUnreachable {
				t.Errorf("run %d (== K): status = %q, want %q", run, got.AccountStatus(), StatusUnreachable)
			}
			if got.StatusReason == "" || got.StatusChangedAt == "" {
				t.Errorf("run %d: transition must record reason+changedAt, got %+v", run, got)
			}
		}
		acct = *got // carry the durable counter into the next run
	}
}

// One success resets the counter — K means *consecutive*, not cumulative.
func TestApplyProbes_SuccessResetsFailureCounter(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	const id = "111111111111"
	existing := []Account{{AccountID: id, Status: StatusActive, ConsecutiveFailures: 5}}
	probes := []ProbeResult{{AccountID: id, Reachable: true, LiveInstances: 0}}

	changed := ApplyProbes(existing, probes, t0, pol)
	got := find(changed, id)
	if got == nil {
		t.Fatal("expected a change row (counter reset + lastSeenAt stamp)")
	}
	if got.ConsecutiveFailures != 0 {
		t.Errorf("consecutiveFailures = %d, want 0 after a successful probe", got.ConsecutiveFailures)
	}
	if got.LastSeenAt != rfc(t0) {
		t.Errorf("lastSeenAt = %q, want %q", got.LastSeenAt, rfc(t0))
	}
	if got.AccountStatus() != StatusActive {
		t.Errorf("status = %q, want active", got.AccountStatus())
	}
}

// Dormancy: reachable AND empty for N. The transition needs a working probe,
// which is what makes emptiness provable rather than assumed.
func TestApplyProbes_DormantAfterN(t *testing.T) {
	pol := DefaultLifecyclePolicy() // N=30d
	const id = "111111111111"

	tests := []struct {
		name       string
		emptySince time.Duration // how long ago the last instance was seen
		want       string
	}{
		{"one day empty is not dormant", 24 * time.Hour, StatusActive},
		{"29 days empty is not dormant", 29 * 24 * time.Hour, StatusActive},
		{"exactly N is dormant", 30 * 24 * time.Hour, StatusDormant},
		{"well past N is dormant", 90 * 24 * time.Hour, StatusDormant},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			existing := []Account{{
				AccountID:      id,
				Status:         StatusActive,
				LastInstanceAt: rfc(t0.Add(-tc.emptySince)),
			}}
			probes := []ProbeResult{{AccountID: id, Reachable: true, LiveInstances: 0}}
			changed := ApplyProbes(existing, probes, t0, pol)
			got := find(changed, id)
			if got == nil {
				// No change row means no transition; only valid when we expected none.
				if tc.want != StatusActive {
					t.Fatalf("expected a transition to %q, got no change", tc.want)
				}
				return
			}
			if got.AccountStatus() != tc.want {
				t.Errorf("status = %q, want %q", got.AccountStatus(), tc.want)
			}
		})
	}
}

// A never-used account must not be instantly dormant: absent evidence is not
// evidence of absence, so the N-day clock starts at first observation.
func TestApplyProbes_NeverUsedAccountSeedsClockNotDormant(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	const id = "111111111111"
	existing := []Account{{AccountID: id, Status: StatusActive}} // no LastInstanceAt
	probes := []ProbeResult{{AccountID: id, Reachable: true, LiveInstances: 0}}

	changed := ApplyProbes(existing, probes, t0, pol)
	got := find(changed, id)
	if got == nil {
		t.Fatal("expected a change row seeding lastInstanceAt")
	}
	if got.AccountStatus() != StatusActive {
		t.Errorf("status = %q, want active — a freshly onboarded account must not be dormant on no evidence", got.AccountStatus())
	}
	if got.LastInstanceAt != rfc(t0) {
		t.Errorf("lastInstanceAt = %q, want the observation time %q", got.LastInstanceAt, rfc(t0))
	}
	// And it does become dormant N later, from that seeded clock.
	later := ApplyProbes([]Account{*got}, probes, t0.Add(pol.DormantAfter), pol)
	if g := find(later, id); g == nil || g.AccountStatus() != StatusDormant {
		t.Errorf("after N from the seeded clock, want dormant; got %+v", g)
	}
}

// Recovery: a dormant or unreachable account that runs a spore again is active.
func TestApplyProbes_RecoveryToActive(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	for _, from := range []string{StatusDormant, StatusUnreachable} {
		t.Run(from, func(t *testing.T) {
			const id = "111111111111"
			existing := []Account{{AccountID: id, Status: from, ConsecutiveFailures: 9}}
			probes := []ProbeResult{{AccountID: id, Reachable: true, LiveInstances: 2}}
			got := find(ApplyProbes(existing, probes, t0, pol), id)
			if got == nil {
				t.Fatal("expected a recovery change row")
			}
			if got.AccountStatus() != StatusActive {
				t.Errorf("status = %q, want active after observing a live instance", got.AccountStatus())
			}
			if got.ConsecutiveFailures != 0 {
				t.Errorf("consecutiveFailures = %d, want 0", got.ConsecutiveFailures)
			}
			if got.LastInstanceAt != rfc(t0) {
				t.Errorf("lastInstanceAt = %q, want %q", got.LastInstanceAt, rfc(t0))
			}
		})
	}
}

// Offboarded is a human decision; no inferred transition may overwrite it.
// Otherwise a deliberately deprovisioned account silently returns to active the
// next time a leftover instance is spotted.
func TestApplyProbes_OffboardedIsNeverAutoRevived(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	const id = "111111111111"

	cases := []struct {
		name  string
		probe ProbeResult
		other ProbeResult
	}{
		{"live instance does not revive", ProbeResult{AccountID: id, Reachable: true, LiveInstances: 3}, ProbeResult{AccountID: "999999999999", Reachable: true}},
		{"empty+reachable does not re-dormant", ProbeResult{AccountID: id, Reachable: true, LiveInstances: 0}, ProbeResult{AccountID: "999999999999", Reachable: true}},
		{"failure does not mark unreachable", ProbeResult{AccountID: id, Reachable: false}, ProbeResult{AccountID: "999999999999", Reachable: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			existing := []Account{{
				AccountID:      id,
				Status:         StatusOffboarded,
				LastInstanceAt: rfc(t0.Add(-365 * 24 * time.Hour)), // long past N
			}}
			// K failures already banked, so the unreachable branch would fire but for
			// the offboarded guard.
			existing[0].ConsecutiveFailures = pol.FailuresBeforeUnreachable
			changed := ApplyProbes(existing, []ProbeResult{tc.probe, tc.other}, t0, pol)
			if got := find(changed, id); got != nil && got.AccountStatus() != StatusOffboarded {
				t.Errorf("status = %q, want %q to survive — offboarding is a stated human decision", got.AccountStatus(), StatusOffboarded)
			}
		})
	}
}

// An account probed but absent from the registry starts its lifecycle from the
// observation: REAPER_ROLE_ARNS predates the registry, so accounts exist in
// config that were never phone-home registered.
func TestApplyProbes_UnknownAccountGetsSeededRow(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	const id = "444444444444"
	changed := ApplyProbes(nil, []ProbeResult{{AccountID: id, Reachable: true, LiveInstances: 1}}, t0, pol)
	got := find(changed, id)
	if got == nil {
		t.Fatal("expected a seeded row for an account absent from the registry")
	}
	if got.AccountID != id || got.LastSeenAt != rfc(t0) {
		t.Errorf("seeded row = %+v, want accountId %q and lastSeenAt %q", got, id, rfc(t0))
	}
}

// Only rows that actually changed come back — the reaper writes one UpdateItem per
// returned row, so a no-op round must not generate traffic. A reachable account
// with an instance still updates lastSeenAt/lastInstanceAt, so the genuinely
// unchanged case is an account absent from this run's probes.
func TestApplyProbes_UnprobedAccountsAreNotReturned(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	existing := []Account{
		{AccountID: "111111111111", Status: StatusActive},
		{AccountID: "222222222222", Status: StatusActive},
	}
	changed := ApplyProbes(existing, []ProbeResult{{AccountID: "111111111111", Reachable: true, LiveInstances: 1}}, t0, pol)
	if find(changed, "222222222222") != nil {
		t.Error("an account not probed this run must not be written")
	}
	if find(changed, "111111111111") == nil {
		t.Error("the probed account should be written (lastSeenAt advanced)")
	}
}

// An unparseable lastInstanceAt must not be guessed at — re-stamp, don't
// transition. Acting on a value we can't read is how a healthy account gets
// deprovisioned by a formatting bug.
func TestApplyProbes_UnparseableTimestampDoesNotTransition(t *testing.T) {
	pol := DefaultLifecyclePolicy()
	const id = "111111111111"
	existing := []Account{{AccountID: id, Status: StatusActive, LastInstanceAt: "not-a-timestamp"}}
	got := find(ApplyProbes(existing, []ProbeResult{{AccountID: id, Reachable: true}}, t0, pol), id)
	if got == nil {
		t.Fatal("expected a re-stamp change row")
	}
	if got.AccountStatus() != StatusActive {
		t.Errorf("status = %q, want active — never transition on an unreadable timestamp", got.AccountStatus())
	}
	if got.LastInstanceAt != rfc(t0) {
		t.Errorf("lastInstanceAt = %q, want re-stamped to %q", got.LastInstanceAt, rfc(t0))
	}
}

func TestApplyProbes_NoProbes(t *testing.T) {
	if changed := ApplyProbes([]Account{{AccountID: "111111111111"}}, nil, t0, DefaultLifecyclePolicy()); changed != nil {
		t.Errorf("no probes must decide nothing, got %+v", changed)
	}
}

// A legacy row (written before the lifecycle fields existed) reads as active, so
// no backfill migration is needed.
func TestAccountStatus_LegacyRowIsActive(t *testing.T) {
	legacy := &Account{AccountID: "111111111111", RoleArn: "arn:aws:iam::111111111111:role/x"}
	if legacy.AccountStatus() != StatusActive {
		t.Errorf("legacy row status = %q, want %q", legacy.AccountStatus(), StatusActive)
	}
	if DNSExpiryEligible(legacy) {
		t.Error("a legacy row must NOT be DNS-expiry eligible")
	}
}

// The payoff question: which states permit deleting an account's DNS records.
// Unreachable is the interesting exclusion (spawn#457 trap 2) — it is the state
// we would most want to clean up, and the one where we can no longer verify
// anything, because the deleted role is what we would have verified through.
func TestDNSExpiryEligible(t *testing.T) {
	tests := []struct {
		status string
		want   bool
		why    string
	}{
		{StatusActive, false, "in use"},
		{StatusUnreachable, false, "emptiness unprovable once the role is gone"},
		{StatusDormant, true, "emptiness proven via a working DescribeInstances"},
		{StatusOffboarded, true, "human stated intent"},
		{"", false, "legacy row reads as active"},
		{"some-future-status", false, "unknown states must not authorize deletion"},
	}
	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			if got := DNSExpiryEligible(&Account{AccountID: "111111111111", Status: tc.status}); got != tc.want {
				t.Errorf("DNSExpiryEligible(%q) = %v, want %v (%s)", tc.status, got, tc.want, tc.why)
			}
		})
	}
}

// ── Persistence ──────────────────────────────────────────────────────────────

// UpdateLifecycle must not disturb the registration fields. A PutAccount here
// would clobber a concurrent re-onboard's fresh ExternalId with the stale copy
// this run happened to read.
func TestUpdateLifecycle_PreservesRegistration(t *testing.T) {
	r := newTestRegistry(t)
	ctx := context.Background()

	orig := &Account{
		AccountID:    verified,
		RoleArn:      "arn:aws:iam::123456789012:role/spore-portal-onboard",
		ExternalId:   "original-external-id",
		Region:       "us-west-2",
		RegisteredBy: "arn:aws:iam::123456789012:user/alice",
	}
	if err := r.PutAccount(ctx, orig); err != nil {
		t.Fatalf("PutAccount: %v", err)
	}

	if err := r.UpdateLifecycle(ctx, &Account{
		AccountID:       verified,
		Status:          StatusDormant,
		LastSeenAt:      rfc(t0),
		LastInstanceAt:  rfc(t0.Add(-40 * 24 * time.Hour)),
		StatusReason:    "reachable with zero managed instances",
		StatusChangedAt: rfc(t0),
	}); err != nil {
		t.Fatalf("UpdateLifecycle: %v", err)
	}

	got, err := r.GetAccount(ctx, verified)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.RoleArn != orig.RoleArn || got.ExternalId != orig.ExternalId ||
		got.Region != orig.Region || got.RegisteredBy != orig.RegisteredBy {
		t.Errorf("registration was disturbed: got %+v, want the fields of %+v", got, orig)
	}
	if got.RegisteredAt == "" {
		t.Error("registeredAt was cleared by the lifecycle update")
	}
	if got.AccountStatus() != StatusDormant {
		t.Errorf("status = %q, want %q", got.AccountStatus(), StatusDormant)
	}
	if got.StatusReason == "" || got.StatusChangedAt == "" || got.LastSeenAt == "" || got.LastInstanceAt == "" {
		t.Errorf("lifecycle fields not persisted: %+v", got)
	}
}

// Empty timestamps must be omitted from the update, not written as "" — an empty
// string would replace a real prior value with one that parses as neither a time
// nor "absent", and the dormancy math reads these back.
func TestUpdateLifecycle_DoesNotBlankExistingTimestamps(t *testing.T) {
	r := newTestRegistry(t)
	ctx := context.Background()

	if err := r.PutAccount(ctx, &Account{
		AccountID: verified, RoleArn: "arn:aws:iam::123456789012:role/x",
		ExternalId: "e", Region: "us-east-1",
	}); err != nil {
		t.Fatalf("PutAccount: %v", err)
	}
	if err := r.UpdateLifecycle(ctx, &Account{
		AccountID: verified, Status: StatusActive, LastInstanceAt: rfc(t0),
	}); err != nil {
		t.Fatalf("seed lastInstanceAt: %v", err)
	}
	// A later round with no LastInstanceAt (e.g. an unreachable probe) must leave
	// the seeded value alone.
	if err := r.UpdateLifecycle(ctx, &Account{
		AccountID: verified, Status: StatusActive, ConsecutiveFailures: 2, LastErrorAt: rfc(t0),
	}); err != nil {
		t.Fatalf("UpdateLifecycle: %v", err)
	}

	got, err := r.GetAccount(ctx, verified)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.LastInstanceAt != rfc(t0) {
		t.Errorf("lastInstanceAt = %q, want the preserved %q", got.LastInstanceAt, rfc(t0))
	}
	if got.ConsecutiveFailures != 2 {
		t.Errorf("consecutiveFailures = %d, want 2", got.ConsecutiveFailures)
	}
}

func TestOffboard(t *testing.T) {
	r := newTestRegistry(t)
	ctx := context.Background()
	if err := r.PutAccount(ctx, &Account{
		AccountID: verified, RoleArn: "arn:aws:iam::123456789012:role/x",
		ExternalId: "e", Region: "us-east-1",
	}); err != nil {
		t.Fatalf("PutAccount: %v", err)
	}
	if err := r.Offboard(ctx, verified, "arn:aws:iam::123456789012:user/operator", t0); err != nil {
		t.Fatalf("Offboard: %v", err)
	}
	got, err := r.GetAccount(ctx, verified)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.AccountStatus() != StatusOffboarded {
		t.Errorf("status = %q, want %q", got.AccountStatus(), StatusOffboarded)
	}
	if !DNSExpiryEligible(got) {
		t.Error("an offboarded account should be DNS-expiry eligible (stated intent)")
	}
	// Offboarding must not destroy the audit trail.
	if got.RoleArn == "" || got.RegisteredAt == "" {
		t.Errorf("offboard erased the registration/audit trail: %+v", got)
	}
}

func TestOffboard_RequiresAccountID(t *testing.T) {
	r := newTestRegistry(t)
	if err := r.Offboard(context.Background(), "", "operator", t0); err == nil {
		t.Error("expected an error for an empty accountId")
	}
}

func TestUpdateLifecycle_RequiresAccountID(t *testing.T) {
	r := newTestRegistry(t)
	if err := r.UpdateLifecycle(context.Background(), &Account{Status: StatusActive}); err == nil {
		t.Error("expected an error for an empty accountId")
	}
	if err := r.UpdateLifecycle(context.Background(), nil); err == nil {
		t.Error("expected an error for a nil account")
	}
}

// ListAccounts feeds ApplyProbes; a truncated result would read as "these
// accounts no longer exist", the exact false conclusion this work avoids.
func TestListAccounts(t *testing.T) {
	r := newTestRegistry(t)
	ctx := context.Background()

	ids := []string{"111111111111", "222222222222", "333333333333"}
	for _, id := range ids {
		if err := r.PutAccount(ctx, &Account{
			AccountID: id, RoleArn: "arn:aws:iam::" + id + ":role/x",
			ExternalId: "e", Region: "us-east-1",
		}); err != nil {
			t.Fatalf("PutAccount %s: %v", id, err)
		}
	}
	got, err := r.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(got) != len(ids) {
		t.Fatalf("ListAccounts returned %d rows, want %d", len(got), len(ids))
	}
	seen := map[string]bool{}
	for _, a := range got {
		seen[a.AccountID] = true
		if a.RoleArn == "" {
			t.Errorf("row %s came back without its roleArn", a.AccountID)
		}
	}
	for _, id := range ids {
		if !seen[id] {
			t.Errorf("account %s missing from ListAccounts", id)
		}
	}
}

func TestListAccounts_Empty(t *testing.T) {
	r := newTestRegistry(t)
	got, err := r.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want no rows, got %d", len(got))
	}
}

// End-to-end over the pure machine plus persistence: an account goes quiet, rides
// out K-1 failures, is marked unreachable at K, and is still NOT DNS-eligible.
func TestLifecycle_QuietAccountEndToEnd(t *testing.T) {
	r := newTestRegistry(t)
	ctx := context.Background()
	pol := DefaultLifecyclePolicy()
	const dead = "111111111111"
	const alive = "222222222222"

	for _, id := range []string{dead, alive} {
		if err := r.PutAccount(ctx, &Account{
			AccountID: id, RoleArn: "arn:aws:iam::" + id + ":role/x",
			ExternalId: "e", Region: "us-east-1",
		}); err != nil {
			t.Fatalf("PutAccount %s: %v", id, err)
		}
	}

	for run := 1; run <= pol.FailuresBeforeUnreachable; run++ {
		rows, err := r.ListAccounts(ctx)
		if err != nil {
			t.Fatalf("run %d ListAccounts: %v", run, err)
		}
		changed := ApplyProbes(rows, []ProbeResult{
			{AccountID: dead, Reachable: false},
			{AccountID: alive, Reachable: true, LiveInstances: 1},
		}, t0.Add(time.Duration(run)*10*time.Minute), pol)
		for i := range changed {
			if err := r.UpdateLifecycle(ctx, &changed[i]); err != nil {
				t.Fatalf("run %d UpdateLifecycle %s: %v", run, changed[i].AccountID, err)
			}
		}
	}

	got, err := r.GetAccount(ctx, dead)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.AccountStatus() != StatusUnreachable {
		t.Errorf("status = %q, want %q after K=%d failures", got.AccountStatus(), StatusUnreachable, pol.FailuresBeforeUnreachable)
	}
	if got.ConsecutiveFailures != pol.FailuresBeforeUnreachable {
		t.Errorf("consecutiveFailures = %d, want %d (the counter must survive across runs)", got.ConsecutiveFailures, pol.FailuresBeforeUnreachable)
	}
	if DNSExpiryEligible(got) {
		t.Error("an unreachable account must NOT be DNS-expiry eligible — emptiness is unprovable once the role is gone")
	}
	// The still-live account stays active and keeps its registration.
	liveRow, err := r.GetAccount(ctx, alive)
	if err != nil {
		t.Fatalf("GetAccount alive: %v", err)
	}
	if liveRow.AccountStatus() != StatusActive || liveRow.ExternalId != "e" {
		t.Errorf("live account row = %+v, want active with its registration intact", liveRow)
	}
}
