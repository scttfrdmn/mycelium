// lifecycle.go — the account deprovisioning state machine (spawn#457).
//
// The problem it solves: onboarding was one-way. Nothing ever concluded "this
// account is gone", so every artifact left behind was permanent — and one of them
// (a Route53 A-record whose public IP has returned to the EC2 pool) eventually
// resolves to a stranger's instance. That is the hazard driving the whole design.
//
// The signal is free: the reaper already assumes every account's role every 10
// minutes, and that probe distinguishes exactly the states that matter — the role
// is gone (the customer uninstalled) versus the role works but the account is
// empty (dormant but reachable). No new API and no customer action beyond the
// uninstall gesture they would already make.
//
// The whole file is pure: no AWS, no clock, no I/O. The reaper supplies probe
// results and `now`; this decides. That is deliberate, because the interesting
// parts are the two refusals — and a refusal is only trustworthy if you can test
// it exhaustively without mocking a cloud.
package main

import (
	"fmt"
	"time"
)

// Lifecycle states. The set is small on purpose: each state has to answer one
// question — "is it safe to delete this account's DNS records?" — and only two
// answers are defensible. Yes (we proved the account is empty, or a human said
// so) and no (we could not prove anything).
const (
	// StatusActive — assume-role works. The normal state.
	StatusActive = "active"
	// StatusUnreachable — assume-role has failed K consecutive runs WHILE other
	// accounts succeeded. Almost certainly an uninstall (deleting the role is the
	// natural uninstall gesture), but not provable, so this state deletes nothing.
	// Its job is to stop counting the account into the reaper's Errors total —
	// that field is supposed to mean "investigate this", and it is worthless once
	// it contains a permanent, expected failure.
	StatusUnreachable = "unreachable"
	// StatusDormant — assume-role WORKS and the account has had zero managed
	// instances for N. The only inferred state where DNS expiry is safe, precisely
	// because reachability is what lets DescribeInstances prove emptiness.
	StatusDormant = "dormant"
	// StatusOffboarded — a human said so. Intent is stated rather than inferred,
	// so this state may delete records.
	StatusOffboarded = "offboarded"
)

// AccountStatus returns the effective status, treating a row written before the
// lifecycle fields existed ("") as active. Callers must use this rather than
// reading .Status directly, or every legacy row looks like an unknown state.
func (a *Account) AccountStatus() string {
	if a == nil {
		return ""
	}
	if a.Status == "" {
		return StatusActive
	}
	return a.Status
}

// ProbeResult is one account's outcome from one reaper run: did the assume-role
// work, and if so, did the account hold any live managed instances.
type ProbeResult struct {
	AccountID string
	// Reachable is whether the assume-role + DescribeInstances succeeded. A false
	// here is the only evidence of uninstall we ever get, and it is weak evidence.
	Reachable bool
	// LiveInstances is how many live managed instances were found. Meaningful only
	// when Reachable — a zero from an unreachable probe proves nothing, because we
	// did not look and could not have.
	LiveInstances int
}

// LifecyclePolicy is the K/N tuning from spawn#457.
type LifecyclePolicy struct {
	// FailuresBeforeUnreachable (K) — consecutive failed runs before an account is
	// marked unreachable. 6 runs at the default rate(10 minutes) is one hour, long
	// enough to ride out a transient STS/API blip.
	FailuresBeforeUnreachable int
	// DormantAfter (N) — how long an account must be reachable-and-empty before it
	// is dormant. 30 days, so a researcher between jobs is not deprovisioned
	// mid-project.
	DormantAfter time.Duration
}

// DefaultLifecyclePolicy is K=6 runs (one hour) and N=30 days — the values
// suggested in spawn#457.
func DefaultLifecyclePolicy() LifecyclePolicy {
	return LifecyclePolicy{FailuresBeforeUnreachable: 6, DormantAfter: 30 * 24 * time.Hour}
}

// ApplyProbes is the state machine: given the current registry rows and this
// run's probe results, return only the rows whose lifecycle fields changed.
//
// THE CENTRAL SAFETY RULE (spawn#457 trap 1): if EVERY probe failed, change
// nothing. The reaper's own role ARN embeds a CloudFormation-generated physical
// ID, so recreating the reaper stack changes that suffix and breaks every
// customer's trust policy simultaneously. A state machine that acted on
// assume-role failure alone would then mark the entire customer base unreachable
// because *we* redeployed. This is the same instinct as the DNS sweep's existing
// refusal to delete against a partial live set: an observation that could be
// explained by our own breakage is not evidence about the customer.
//
// now is a parameter rather than a clock read so tests can drive the N-day
// transitions without sleeping.
func ApplyProbes(existing []Account, probes []ProbeResult, now time.Time, pol LifecyclePolicy) []Account {
	if len(probes) == 0 {
		return nil
	}
	// Correlated-failure guard. Note this tests probes, not existing rows: the
	// question is whether OUR probing worked at all, and a single-account
	// deployment whose one probe failed is exactly as uninformative as a
	// hundred-account one where all hundred failed.
	anyReachable := false
	for _, p := range probes {
		if p.Reachable {
			anyReachable = true
			break
		}
	}
	if !anyReachable {
		return nil
	}

	byID := make(map[string]Account, len(existing))
	for _, a := range existing {
		byID[a.AccountID] = a
	}
	stamp := now.UTC().Format(time.RFC3339)

	var changed []Account
	for _, p := range probes {
		// An account being probed but absent from the registry is not an error:
		// REAPER_ROLE_ARNS predates the registry, so early accounts were onboarded
		// by hand-editing config. Start its lifecycle from what we just observed.
		acct, known := byID[p.AccountID]
		if !known {
			acct = Account{AccountID: p.AccountID}
		}
		before := acct

		if !p.Reachable {
			// Failed, and at least one other account succeeded — so this is about
			// this account, not about us.
			acct.ConsecutiveFailures++
			acct.LastErrorAt = stamp
			// Offboarded is excluded alongside unreachable itself: an offboarded
			// account's role being gone is the EXPECTED outcome of deprovisioning, so
			// relabeling it "unreachable" would overwrite a stated human decision
			// with a weaker inferred one — and would strip the DNS-expiry eligibility
			// that offboarding deliberately granted.
			if s := acct.AccountStatus(); acct.ConsecutiveFailures >= pol.FailuresBeforeUnreachable &&
				s != StatusUnreachable && s != StatusOffboarded {
				setStatus(&acct, StatusUnreachable, stamp, fmt.Sprintf(
					"assume-role failed %d consecutive runs while other accounts succeeded",
					acct.ConsecutiveFailures))
			}
			if acct != before {
				changed = append(changed, acct)
			}
			continue
		}

		// Reachable: a liveness observation regardless of instance count.
		acct.ConsecutiveFailures = 0
		acct.LastSeenAt = stamp

		switch {
		case p.LiveInstances > 0:
			acct.LastInstanceAt = stamp
			if s := acct.AccountStatus(); s != StatusActive && s != StatusOffboarded {
				// Recovery: a dormant or unreachable account that runs a spore again
				// is active. Never auto-revive an offboarded one — that status was a
				// human decision, and only a human (a re-onboard) should undo it.
				setStatus(&acct, StatusActive, stamp, "live managed instance observed")
			}

		case acct.LastInstanceAt == "":
			// Reachable and empty, but we have never seen an instance here. Seed the
			// clock from this observation rather than treating "no evidence" as
			// "empty since forever" — otherwise a freshly onboarded account is
			// instantly dormant on the strength of nothing at all.
			//
			// Behaviorally redundant, and deliberately kept: the default branch below
			// reaches the same result, because time.Parse of "" errors and lands in its
			// re-stamp path. Mutation testing confirms no test can tell the two apart.
			// It stays because "never observed" and "observed but unreadable" are
			// different facts that happen to share a remedy, and leaning on the empty
			// string failing to parse would make this correct only by accident.
			acct.LastInstanceAt = stamp

		default:
			last, err := time.Parse(time.RFC3339, acct.LastInstanceAt)
			if err != nil {
				// Unparseable timestamp: re-stamp rather than guess. Refusing to act
				// on a value we cannot read is the same instinct as the rest of this file.
				acct.LastInstanceAt = stamp
				break
			}
			if now.UTC().Sub(last) >= pol.DormantAfter {
				if s := acct.AccountStatus(); s != StatusDormant && s != StatusOffboarded {
					setStatus(&acct, StatusDormant, stamp, fmt.Sprintf(
						"reachable with zero managed instances since %s", acct.LastInstanceAt))
				}
			}
		}

		if acct != before {
			changed = append(changed, acct)
		}
	}
	return changed
}

func setStatus(a *Account, status, stamp, reason string) {
	a.Status = status
	a.StatusReason = reason
	a.StatusChangedAt = stamp
}

// DNSExpiryEligible reports whether it is safe to delete this account's Route53
// records — the one question the state machine exists to answer.
//
// Only dormant (emptiness PROVEN via a working DescribeInstances) and offboarded
// (a human stated intent) qualify. Unreachable deliberately does not, and that is
// spawn#457's trap 2: the moment we would most like to clean up is the moment we
// have lost the ability to verify, because the deleted role is what we would have
// verified through. Records under an unreachable account stay, and surface via the
// reaper's report-only unmanaged-subdomain signal for a human to resolve.
func DNSExpiryEligible(a *Account) bool {
	switch a.AccountStatus() {
	case StatusDormant, StatusOffboarded:
		return true
	default:
		return false
	}
}
