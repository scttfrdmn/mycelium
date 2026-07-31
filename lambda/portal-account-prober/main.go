// portal-account-prober — the caller ApplyProbes was written for (spawn#457).
//
// Runs on a schedule in the infra account (966362334030). Each run: scan the
// account registry, assume each account's onboarding role, count spawn-managed
// instances across every region, and hand the results to
// accountlifecycle.ApplyProbes, which decides the lifecycle transitions. Only the
// rows it says changed are written back.
//
// WHY A SEPARATE LAMBDA, and not the ttl-reaper that already assumes into customer
// accounts every 10 minutes: the reaper assumes `spawn-ttl-reaper-ec2`, created by
// a manual CloudFormation deploy and listed in REAPER_ROLE_ARNS. Onboarding creates
// a DIFFERENT role, `spore-portal-onboard`, which trusts only the phone-home
// Lambda's role. So the reaper cannot assume the roles the registry knows about —
// its probe covers a different, hand-maintained set of accounts. Reading the
// registry from the reaper would have meant probing accounts it has no credentials
// for and recording their unreachability as fact.
//
// And not the phone-home Lambda itself: that function is internet-facing under a
// Function URL. Granting it sts:AssumeRole into every customer account would put
// "assume into any onboarded account" one bug away from the public edge. The prober
// has no URL, no public trigger, and does nothing but read the registry and count
// instances.
//
// THE ROLLOUT HAZARD this is built around: `spore-portal-onboard` trust policies
// written before this Lambda existed name only the phone-home role, so they will
// refuse the prober — and STS reports that refusal with the same AccessDenied it
// uses for a role that was deleted. Acting on the first reading when the second is
// true deprovisions healthy accounts wholesale, and the correlated-failure guard
// cannot catch it (it fires only when EVERY probe fails, and this affects a
// subset). accountlifecycle resolves it with the registry's own LastSeenAt: a
// denial is evidence only for an account that has succeeded at least once. Every
// account onboarded before the template update is therefore left strictly alone
// until it re-onboards.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spore-host/spore-host/lambda/accountlifecycle"
)

// defaultRegions mirrors the ttl-reaper's set — the regions the release workflow
// publishes spored to, i.e. where a spore can actually land. Override with
// PROBER_REGIONS.
//
// This list is a correctness input, not just a cost knob: emptiness is established
// by finding nothing across it, so a region missing from here is a region whose
// instances are invisible to dormancy. Shrinking it to save API calls trades
// directly against the risk of calling a busy account dormant.
var defaultRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"ca-central-1",
	"eu-west-1", "eu-west-2", "eu-central-1",
	"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
}

// sessionPolicy is attached to every assume-role the prober performs, capping the
// resulting credentials at the single call this function makes. See the assume-role
// site below for why that cap is load-bearing rather than decorative.
//
// Inline rather than built with a marshaller: this is the security boundary, and it
// should be readable as exactly what STS receives.
const sessionPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ProbeOnly",
      "Effect": "Allow",
      "Action": "ec2:DescribeInstances",
      "Resource": "*"
    }
  ]
}`

// registryAPI is the registry surface the run loop uses. An interface rather than
// the concrete *accountlifecycle.Registry so the run loop — which is where the
// decision to WRITE lives — can be tested against a fake that records every call.
// The riskiest assertion in this package is about writes that must not happen, and
// that is only checkable if the writer is observable.
type registryAPI interface {
	ListAccounts(ctx context.Context) ([]accountlifecycle.Account, error)
	UpdateLifecycle(ctx context.Context, acct *accountlifecycle.Account) error
}

type prober struct {
	reg     registryAPI
	regions []string
	policy  accountlifecycle.LifecyclePolicy
	dryRun  bool
	mk      regionalEC2
}

var p *prober

// Summary is the run report (also the Lambda's return value, so it lands in the
// invocation log and any downstream metric filter).
type Summary struct {
	Accounts  int    `json:"accounts"`
	Reachable int    `json:"reachable"`
	Denied    int    `json:"denied"`
	Failed    int    `json:"failed"`
	Partial   int    `json:"partial_region_coverage"`
	Changed   int    `json:"changed"`
	Written   int    `json:"written"`
	Errors    int    `json:"errors"`
	Refused   bool   `json:"refused_correlated_failure"`
	Duration  string `json:"duration"`
}

func init() {
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}
	stsClient := sts.NewFromConfig(cfg)

	pol := accountlifecycle.DefaultLifecyclePolicy()
	if v := os.Getenv("PROBER_FAILURES_BEFORE_UNREACHABLE"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			pol.FailuresBeforeUnreachable = k
		} else {
			log.Printf("ignoring PROBER_FAILURES_BEFORE_UNREACHABLE=%q: %v", v, err)
		}
	}
	if v := os.Getenv("PROBER_DORMANT_AFTER"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			pol.DormantAfter = d
		} else {
			log.Printf("ignoring PROBER_DORMANT_AFTER=%q: %v", v, err)
		}
	}

	p = &prober{
		reg:     accountlifecycle.NewRegistry(cfg),
		regions: parseRegions(os.Getenv("PROBER_REGIONS")),
		policy:  pol,
		dryRun:  os.Getenv("PROBER_DRY_RUN") == "true",
		mk: func(ctx context.Context, acct accountlifecycle.Account, region string) (ec2API, error) {
			c := cfg.Copy()
			c.Region = region
			// The ExternalId is the per-account confused-deputy guard the onboarding
			// template pinned into the role's trust policy; without it the assume is
			// refused. Retrieve() is called eagerly so an assume-role refusal surfaces
			// here as an error rather than later inside DescribeInstances, where it
			// would be indistinguishable from an EC2-level denial.
			//
			// The session policy is the important part. spore-portal-onboard is a
			// LAUNCH role — RunInstances, TerminateInstances, PassRole — because that is
			// what the portal needs, and a trust policy governs who may assume, not what
			// they may then do. So naming the prober as a trusted principal would
			// otherwise hand it the full launch capability to do one read call with.
			// Effective permissions are the INTERSECTION of the role's policy and this
			// session policy, so these credentials cannot launch or terminate anything
			// even if this function is compromised — which is what makes it honest to
			// ask customers for the grant rather than shipping a second read-only role.
			prov := stscreds.NewAssumeRoleProvider(stsClient, acct.RoleArn, func(o *stscreds.AssumeRoleOptions) {
				o.ExternalID = aws.String(acct.ExternalId)
				o.RoleSessionName = "portal-account-prober"
				o.Policy = aws.String(sessionPolicy)
			})
			if _, err := prov.Retrieve(ctx); err != nil {
				return nil, err
			}
			c.Credentials = aws.NewCredentialsCache(prov)
			return ec2.NewFromConfig(c), nil
		},
	}
	log.Printf("portal-account-prober initialized (regions=%v, K=%d runs, N=%s, dry-run=%t)",
		p.regions, p.policy.FailuresBeforeUnreachable, p.policy.DormantAfter, p.dryRun)
}

func parseRegions(env string) []string {
	if strings.TrimSpace(env) == "" {
		return defaultRegions
	}
	var out []string
	for _, r := range strings.Split(env, ",") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return defaultRegions
	}
	return out
}

func handler(ctx context.Context) (Summary, error) {
	return p.run(ctx, time.Now().UTC())
}

func (pr *prober) run(ctx context.Context, now time.Time) (Summary, error) {
	start := time.Now()
	var sum Summary

	accounts, err := pr.reg.ListAccounts(ctx)
	if err != nil {
		// Fail loudly rather than proceeding on a partial set. A short account list is
		// indistinguishable from accounts that no longer exist — except that we would
		// simply not probe the missing ones, and ApplyProbes ignores unprobed accounts.
		// So this is safe, but it is still a broken run and must not report success.
		return sum, fmt.Errorf("list accounts: %w", err)
	}
	sum.Accounts = len(accounts)
	if len(accounts) == 0 {
		sum.Duration = time.Since(start).String()
		log.Print("no registered accounts; nothing to probe")
		return sum, nil
	}

	probes := make([]accountlifecycle.ProbeResult, 0, len(accounts))
	for _, acct := range accounts {
		if acct.RoleArn == "" || acct.ExternalId == "" {
			// A row with no credentials to use is not an unreachable account, it is an
			// incomplete registration. Skipping it keeps it out of the probe set, and
			// ApplyProbes never touches an account it was not given a probe for.
			log.Printf("account %s: incomplete registration (roleArn/externalId missing); skipping", acct.AccountID)
			sum.Errors++
			continue
		}
		res := probeAccount(ctx, acct, pr.regions, pr.mk)
		switch {
		case res.Reachable:
			sum.Reachable++
			if res.EmptinessUnproven {
				sum.Partial++
			}
		case res.AssumeRoleDenied:
			sum.Denied++
		default:
			sum.Failed++
		}
		probes = append(probes, res)
	}

	changed := accountlifecycle.ApplyProbes(accounts, probes, now, pr.policy)
	sum.Changed = len(changed)

	// Surface the correlated-failure refusal explicitly. ApplyProbes returning
	// nothing on an all-failed round is correct, but silently correct — and a run
	// that decides nothing because OUR probing is broken looks exactly like a run
	// with nothing to do. Alarm on this field, not on Errors.
	if sum.Reachable == 0 {
		sum.Refused = true
		log.Printf("REFUSING to conclude anything: 0 of %d accounts were reachable this run "+
			"(denied=%d failed=%d). An observation explainable by our own breakage is not "+
			"evidence about customers. Investigate the prober's own credentials/trust first.",
			len(probes), sum.Denied, sum.Failed)
	}

	for i := range changed {
		row := &changed[i]
		if pr.dryRun {
			log.Printf("WOULD update %s → status=%q failures=%d reason=%q",
				row.AccountID, row.AccountStatus(), row.ConsecutiveFailures, row.StatusReason)
			continue
		}
		if err := pr.reg.UpdateLifecycle(ctx, row); err != nil {
			log.Printf("account %s: update lifecycle: %v", row.AccountID, err)
			sum.Errors++
			continue
		}
		sum.Written++
		if row.StatusChangedAt != "" {
			log.Printf("account %s → %s (%s)", row.AccountID, row.AccountStatus(), row.StatusReason)
		}
	}

	sum.Duration = time.Since(start).String()
	log.Printf("probe run complete: %+v", sum)
	return sum, nil
}

func main() {
	lambda.Start(handler)
}
