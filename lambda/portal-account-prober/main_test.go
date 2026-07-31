package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/spore-host/spore-host/lambda/accountlifecycle"
)

var t0 = time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

// fakeRegistry records every write, because the sharpest assertions in this package
// are about writes that must NOT happen.
type fakeRegistry struct {
	rows     []accountlifecycle.Account
	listErr  error
	writes   []accountlifecycle.Account
	writeErr error
}

func (f *fakeRegistry) ListAccounts(ctx context.Context) ([]accountlifecycle.Account, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.rows, nil
}

func (f *fakeRegistry) UpdateLifecycle(ctx context.Context, acct *accountlifecycle.Account) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.writes = append(f.writes, *acct)
	return nil
}

// wroteFor returns the MOST RECENT write for an account — across a multi-run test
// the last one is the account's current state, and the first is merely history.
func (f *fakeRegistry) wroteFor(id string) *accountlifecycle.Account {
	for i := len(f.writes) - 1; i >= 0; i-- {
		if f.writes[i].AccountID == id {
			return &f.writes[i]
		}
	}
	return nil
}

func registered(id string) accountlifecycle.Account {
	return accountlifecycle.Account{
		AccountID:  id,
		RoleArn:    "arn:aws:iam::" + id + ":role/spore-portal-onboard",
		ExternalId: "e-" + id,
		Region:     "us-east-1",
	}
}

func newProber(reg registryAPI, mk regionalEC2) *prober {
	return &prober{
		reg:     reg,
		regions: threeRegions,
		policy:  accountlifecycle.DefaultLifecyclePolicy(),
		mk:      mk,
	}
}

// The happy path: a reachable account with instances gets its liveness stamped.
func TestRun_HealthyAccountStampsLiveness(t *testing.T) {
	reg := &fakeRegistry{rows: []accountlifecycle.Account{registered("111111111111")}}
	s := regionScript{counts: map[string]int{"us-east-1": 1}}
	sum, err := newProber(reg, s.mk).run(context.Background(), t0)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if sum.Reachable != 1 || sum.Written != 1 || sum.Refused {
		t.Errorf("summary = %+v, want 1 reachable, 1 written, not refused", sum)
	}
	got := reg.wroteFor("111111111111")
	if got == nil || got.LastSeenAt != t0.Format(time.RFC3339) {
		t.Errorf("wrote %+v, want lastSeenAt %q", got, t0.Format(time.RFC3339))
	}
}

// THE rollout-hazard test, end to end through the prober. Every existing
// `spore-portal-onboard` trust policy was written before this Lambda existed and
// names only the phone-home role, so it will refuse us — with the same AccessDenied
// STS uses for a deleted role. If the prober treated that as evidence, its first
// production runs would march the entire pre-existing customer base to unreachable.
// Sustained past K, alongside a healthy account so the correlated-failure guard is
// NOT what saves them.
func TestRun_PreExistingAccountsSurviveTheProberRollout(t *testing.T) {
	pol := accountlifecycle.DefaultLifecyclePolicy()
	legacy := []string{"111111111111", "222222222222", "333333333333"}

	rows := []accountlifecycle.Account{registered("999999999999")} // onboarded post-update
	for _, id := range legacy {
		rows = append(rows, registered(id)) // no LastSeenAt: never probed successfully
	}
	reg := &fakeRegistry{rows: rows}

	// The new account works; every legacy account refuses us.
	mk := func(ctx context.Context, acct accountlifecycle.Account, region string) (ec2API, error) {
		if acct.AccountID == "999999999999" {
			return &fakeEC2{count: 1}, nil
		}
		return nil, apiErr("AccessDenied")
	}
	pr := newProber(reg, mk)

	for run := 1; run <= pol.FailuresBeforeUnreachable*3; run++ {
		sum, err := pr.run(context.Background(), t0.Add(time.Duration(run)*time.Hour))
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		if sum.Refused {
			t.Fatalf("run %d: the healthy account was reachable, so the correlated-failure guard must NOT be what spares the others", run)
		}
		// Carry writes forward the way DynamoDB would.
		for i := range reg.writes {
			for j := range reg.rows {
				if reg.rows[j].AccountID == reg.writes[i].AccountID {
					w := reg.writes[i]
					reg.rows[j].Status = w.Status
					reg.rows[j].LastSeenAt = w.LastSeenAt
					reg.rows[j].ConsecutiveFailures = w.ConsecutiveFailures
					reg.rows[j].LastInstanceAt = w.LastInstanceAt
				}
			}
		}
	}

	for _, id := range legacy {
		if got := reg.wroteFor(id); got != nil {
			t.Errorf("account %s was written (%+v); an account that never let us in has no baseline, so its denial is our rollout gap and not its uninstall", id, got)
		}
		for _, row := range reg.rows {
			if row.AccountID == id && row.AccountStatus() != accountlifecycle.StatusActive {
				t.Errorf("account %s ended at %q; a healthy pre-existing account must stay active", id, row.AccountStatus())
			}
		}
	}
}

// The other half: an account that HAS succeeded before, then starts refusing, is a
// real uninstall and must be detected. Without this the design reads no signal at all.
func TestRun_UninstallAfterBaselineIsDetected(t *testing.T) {
	pol := accountlifecycle.DefaultLifecyclePolicy()
	gone := registered("111111111111")
	gone.LastSeenAt = t0.Add(-time.Hour).Format(time.RFC3339) // proven baseline
	reg := &fakeRegistry{rows: []accountlifecycle.Account{gone, registered("999999999999")}}

	mk := func(ctx context.Context, acct accountlifecycle.Account, region string) (ec2API, error) {
		if acct.AccountID == "999999999999" {
			return &fakeEC2{count: 1}, nil
		}
		return nil, apiErr("AccessDenied")
	}
	pr := newProber(reg, mk)

	for run := 1; run <= pol.FailuresBeforeUnreachable; run++ {
		if _, err := pr.run(context.Background(), t0.Add(time.Duration(run)*time.Hour)); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		for i := range reg.writes {
			for j := range reg.rows {
				if reg.rows[j].AccountID == reg.writes[i].AccountID {
					reg.rows[j].Status = reg.writes[i].Status
					reg.rows[j].ConsecutiveFailures = reg.writes[i].ConsecutiveFailures
					reg.rows[j].LastSeenAt = reg.writes[i].LastSeenAt
				}
			}
		}
	}
	got := reg.wroteFor("111111111111")
	if got == nil {
		t.Fatal("a denial after a proven success must be recorded")
	}
	if got.AccountStatus() != accountlifecycle.StatusUnreachable {
		t.Errorf("status = %q, want %q after K denials against a proven baseline",
			got.AccountStatus(), accountlifecycle.StatusUnreachable)
	}
}

// A run where nothing was reachable must set Refused and write nothing. This is the
// field to alarm on: a run that decides nothing because OUR credentials broke looks
// identical to a run with nothing to do.
func TestRun_TotalFailureRefusesAndWritesNothing(t *testing.T) {
	reg := &fakeRegistry{rows: []accountlifecycle.Account{
		registered("111111111111"), registered("222222222222"),
	}}
	mk := func(ctx context.Context, acct accountlifecycle.Account, region string) (ec2API, error) {
		return nil, errors.New("sts unreachable")
	}
	sum, err := newProber(reg, mk).run(context.Background(), t0)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !sum.Refused {
		t.Error("want Refused set — this is the alarm signal for our own breakage")
	}
	if sum.Failed != 2 || sum.Reachable != 0 {
		t.Errorf("summary = %+v, want 2 failed / 0 reachable", sum)
	}
	if len(reg.writes) != 0 {
		t.Errorf("wrote %d row(s) on a wholly-failed run: %+v", len(reg.writes), reg.writes)
	}
}

// A partial region sweep must not let a long-empty account go dormant — dormant is
// one of the two states that authorizes deleting the account's DNS records.
func TestRun_PartialSweepDoesNotDeclareDormant(t *testing.T) {
	acct := registered("111111111111")
	acct.LastInstanceAt = t0.Add(-90 * 24 * time.Hour).Format(time.RFC3339) // long past N
	reg := &fakeRegistry{rows: []accountlifecycle.Account{acct}}
	s := regionScript{describeEr: map[string]error{"eu-west-1": apiErr("Throttling")}}

	sum, err := newProber(reg, s.mk).run(context.Background(), t0)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if sum.Partial != 1 {
		t.Errorf("summary = %+v, want the partial-coverage count surfaced", sum)
	}
	if got := reg.wroteFor("111111111111"); got != nil && got.AccountStatus() == accountlifecycle.StatusDormant {
		t.Error("declared dormant off a partial sweep: zero-of-a-partial-set is not zero")
	}
}

// A complete empty sweep past N does reach dormant — the deferral above must not be
// permanent, or dormancy never happens and the DNS records this work exists to
// expire are never eligible.
func TestRun_CompleteEmptySweepReachesDormant(t *testing.T) {
	acct := registered("111111111111")
	acct.LastInstanceAt = t0.Add(-90 * 24 * time.Hour).Format(time.RFC3339)
	reg := &fakeRegistry{rows: []accountlifecycle.Account{acct}}

	if _, err := newProber(reg, regionScript{}.mk).run(context.Background(), t0); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := reg.wroteFor("111111111111")
	if got == nil || got.AccountStatus() != accountlifecycle.StatusDormant {
		t.Fatalf("wrote %+v, want dormant after a complete empty sweep past N", got)
	}
	if !accountlifecycle.DNSExpiryEligible(got) {
		t.Error("a proven-dormant account should be DNS-expiry eligible")
	}
}

// An incomplete registration is not an unreachable account. Probing it with no
// credentials would manufacture a failure and count it toward unreachable.
func TestRun_IncompleteRegistrationIsSkippedNotProbed(t *testing.T) {
	partial := accountlifecycle.Account{AccountID: "111111111111"} // no roleArn/externalId
	reg := &fakeRegistry{rows: []accountlifecycle.Account{partial, registered("999999999999")}}

	probed := map[string]bool{}
	mk := func(ctx context.Context, acct accountlifecycle.Account, region string) (ec2API, error) {
		probed[acct.AccountID] = true
		return &fakeEC2{count: 1}, nil
	}
	sum, err := newProber(reg, mk).run(context.Background(), t0)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if probed["111111111111"] {
		t.Error("an account with no credentials to use must not be probed")
	}
	if got := reg.wroteFor("111111111111"); got != nil {
		t.Errorf("incomplete registration was written: %+v", got)
	}
	if sum.Errors != 1 {
		t.Errorf("errors = %d, want 1 — the bad row should be visible, not silent", sum.Errors)
	}
}

// dry-run decides everything and writes nothing.
func TestRun_DryRunWritesNothing(t *testing.T) {
	acct := registered("111111111111")
	acct.LastInstanceAt = t0.Add(-90 * 24 * time.Hour).Format(time.RFC3339)
	reg := &fakeRegistry{rows: []accountlifecycle.Account{acct}}
	pr := newProber(reg, regionScript{}.mk)
	pr.dryRun = true

	sum, err := pr.run(context.Background(), t0)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if sum.Changed == 0 {
		t.Error("dry-run should still decide")
	}
	if sum.Written != 0 || len(reg.writes) != 0 {
		t.Errorf("dry-run wrote %d row(s)", len(reg.writes))
	}
}

// A failed Scan must surface as an error, not a quiet zero-account success. The run
// is safe either way (ApplyProbes ignores unprobed accounts) but it is still broken.
func TestRun_ListErrorFails(t *testing.T) {
	reg := &fakeRegistry{listErr: errors.New("dynamodb unavailable")}
	if _, err := newProber(reg, regionScript{}.mk).run(context.Background(), t0); err == nil {
		t.Error("expected an error when the registry cannot be read")
	}
}

func TestRun_NoAccounts(t *testing.T) {
	sum, err := newProber(&fakeRegistry{}, regionScript{}.mk).run(context.Background(), t0)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if sum.Accounts != 0 || sum.Refused {
		t.Errorf("summary = %+v, want an empty non-refused run", sum)
	}
}

// The session policy is what makes it honest to ask customers to trust the prober
// with a role that can RunInstances: effective permissions are the intersection, so
// these credentials can only read. A malformed policy fails every assume-role at
// runtime, and an over-broad one silently removes the cap — neither is visible
// without checking the document itself.
func TestSessionPolicy_IsValidAndReadOnly(t *testing.T) {
	var doc struct {
		Version   string `json:"Version"`
		Statement []struct {
			Sid      string `json:"Sid"`
			Effect   string `json:"Effect"`
			Action   any    `json:"Action"`
			Resource any    `json:"Resource"`
		} `json:"Statement"`
	}
	if err := json.Unmarshal([]byte(sessionPolicy), &doc); err != nil {
		t.Fatalf("session policy is not valid JSON, so every assume-role would fail: %v", err)
	}
	if doc.Version != "2012-10-17" {
		t.Errorf("Version = %q, want 2012-10-17", doc.Version)
	}
	if len(doc.Statement) != 1 {
		t.Fatalf("want exactly one statement, got %d — every addition widens what a compromised prober can do", len(doc.Statement))
	}
	st := doc.Statement[0]
	if st.Effect != "Allow" {
		t.Errorf("Effect = %q, want Allow", st.Effect)
	}
	// A single string, not a list: a list is where extra actions get appended.
	act, ok := st.Action.(string)
	if !ok {
		t.Fatalf("Action = %#v, want the single string \"ec2:DescribeInstances\"", st.Action)
	}
	if act != "ec2:DescribeInstances" {
		t.Errorf("Action = %q — the prober makes exactly one call, and this cap is what keeps a launch-capable role from being launch-capable in its hands", act)
	}
}

// A write failure is counted, and does not abort the remaining accounts.
func TestRun_WriteErrorIsCountedNotFatal(t *testing.T) {
	reg := &fakeRegistry{
		rows:     []accountlifecycle.Account{registered("111111111111"), registered("222222222222")},
		writeErr: errors.New("throttled"),
	}
	s := regionScript{counts: map[string]int{"us-east-1": 1}}
	sum, err := newProber(reg, s.mk).run(context.Background(), t0)
	if err != nil {
		t.Fatalf("run should not abort on a write failure: %v", err)
	}
	if sum.Errors != 2 || sum.Written != 0 {
		t.Errorf("summary = %+v, want 2 errors / 0 written", sum)
	}
}
