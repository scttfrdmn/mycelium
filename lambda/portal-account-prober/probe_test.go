package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/spore-host/spore-host/lambda/accountlifecycle"
)

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeEC2 struct {
	count int
	err   error
}

func (f *fakeEC2) DescribeInstances(ctx context.Context, in *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	insts := make([]ec2types.Instance, f.count)
	return &ec2.DescribeInstancesOutput{
		Reservations: []ec2types.Reservation{{Instances: insts}},
	}, nil
}

// pagedEC2 returns one full page (with a NextToken, so the paginator asks again)
// and then fails. This is the only shape that can distinguish discarding a partial
// count from returning it — a fake that fails on the FIRST page has nothing
// accumulated yet, so both behaviours look identical.
type pagedEC2 struct {
	firstPage int
	calls     int
}

func (f *pagedEC2) DescribeInstances(ctx context.Context, in *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	f.calls++
	if f.calls == 1 {
		return &ec2.DescribeInstancesOutput{
			Reservations: []ec2types.Reservation{{Instances: make([]ec2types.Instance, f.firstPage)}},
			NextToken:    aws.String("page-2"),
		}, nil
	}
	return nil, apiErr("Throttling")
}

// apiErr builds a real smithy.APIError so isDenied is exercised through the same
// errors.As path production takes, not a string match.
func apiErr(code string) error {
	return &smithy.GenericAPIError{Code: code, Message: code}
}

// regionScript drives per-region behaviour: a region maps to an instance count, or
// to an error (on the assume-role or on DescribeInstances).
type regionScript struct {
	counts     map[string]int
	credErr    map[string]error
	describeEr map[string]error
}

func (s regionScript) mk(ctx context.Context, acct accountlifecycle.Account, region string) (ec2API, error) {
	if err, ok := s.credErr[region]; ok {
		return nil, err
	}
	return &fakeEC2{count: s.counts[region], err: s.describeEr[region]}, nil
}

var threeRegions = []string{"us-east-1", "us-west-2", "eu-west-1"}

func testAccount() accountlifecycle.Account {
	return accountlifecycle.Account{
		AccountID:  "111111111111",
		RoleArn:    "arn:aws:iam::111111111111:role/spore-portal-onboard",
		ExternalId: "high-entropy",
	}
}

// ── isDenied ─────────────────────────────────────────────────────────────────

// The AccessDenied family must be recognized through errors.As on a real API error.
// A miss here silently converts the uninstall signal into a generic failure — the
// account still trends unreachable, but via the path that has no baseline check.
func TestIsDenied(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"AccessDenied (STS assume-role)", apiErr("AccessDenied"), true},
		{"AccessDeniedException", apiErr("AccessDeniedException"), true},
		{"UnauthorizedOperation (EC2)", apiErr("UnauthorizedOperation"), true},
		{"Throttling is NOT a denial", apiErr("Throttling"), false},
		{"RequestLimitExceeded is NOT a denial", apiErr("RequestLimitExceeded"), false},
		{"a plain error is not a denial", errors.New("dial tcp: timeout"), false},
		{"wrapped denial is still a denial", errors.Join(errors.New("ctx"), apiErr("AccessDenied")), true},
		{"nil", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDenied(tc.err); got != tc.want {
				t.Errorf("isDenied(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// ── probeAccount ─────────────────────────────────────────────────────────────

func TestProbeAccount_HealthyAccountSumsAcrossRegions(t *testing.T) {
	s := regionScript{counts: map[string]int{"us-east-1": 2, "eu-west-1": 3}}
	res := probeAccount(context.Background(), testAccount(), threeRegions, s.mk)

	if !res.Reachable {
		t.Error("want reachable")
	}
	if res.LiveInstances != 5 {
		t.Errorf("liveInstances = %d, want 5 summed across regions", res.LiveInstances)
	}
	if res.EmptinessUnproven {
		t.Error("every region answered, so emptiness IS proven")
	}
	if res.AssumeRoleDenied {
		t.Error("no denial occurred")
	}
}

// A complete sweep finding nothing is the only thing that may prove emptiness —
// this is the input dormancy is allowed to act on.
func TestProbeAccount_CompleteEmptySweepProvesEmptiness(t *testing.T) {
	res := probeAccount(context.Background(), testAccount(), threeRegions, regionScript{}.mk)
	if !res.Reachable || res.LiveInstances != 0 {
		t.Fatalf("want reachable with 0 instances, got %+v", res)
	}
	if res.EmptinessUnproven {
		t.Error("all regions answered zero — emptiness is proven and dormancy may proceed")
	}
}

// The load-bearing case: one region throttles, so the zero we collected is a zero
// of a PARTIAL set. Reporting it as proven emptiness is what would let a throttle
// deprovision a busy account.
func TestProbeAccount_PartialSweepFlagsEmptinessUnproven(t *testing.T) {
	s := regionScript{describeEr: map[string]error{"eu-west-1": apiErr("Throttling")}}
	res := probeAccount(context.Background(), testAccount(), threeRegions, s.mk)

	if !res.Reachable {
		t.Error("two regions answered, so the account IS reachable")
	}
	if !res.EmptinessUnproven {
		t.Error("a region failed, so this zero is zero-of-a-partial-set and must not prove emptiness")
	}
}

// Same for a region whose credentials failed for a non-denial reason.
func TestProbeAccount_PartialCredentialFailureFlagsUnproven(t *testing.T) {
	s := regionScript{credErr: map[string]error{"us-west-2": errors.New("network unreachable")}}
	res := probeAccount(context.Background(), testAccount(), threeRegions, s.mk)
	if !res.Reachable || !res.EmptinessUnproven {
		t.Errorf("want reachable + unproven, got %+v", res)
	}
}

// A partial sweep that DID find instances still reports what it found: positive
// evidence is not weakened by incomplete coverage.
func TestProbeAccount_PartialSweepStillReportsFoundInstances(t *testing.T) {
	s := regionScript{
		counts:     map[string]int{"us-east-1": 4},
		describeEr: map[string]error{"eu-west-1": apiErr("Throttling")},
	}
	res := probeAccount(context.Background(), testAccount(), threeRegions, s.mk)
	if res.LiveInstances != 4 {
		t.Errorf("liveInstances = %d, want 4", res.LiveInstances)
	}
	if !res.EmptinessUnproven {
		t.Error("coverage was still partial")
	}
}

// An assume-role refusal is one observation, not eleven: it short-circuits rather
// than re-asking every region for the same AccessDenied.
func TestProbeAccount_DeniedShortCircuitsAllRegions(t *testing.T) {
	attempts := 0
	mk := func(ctx context.Context, acct accountlifecycle.Account, region string) (ec2API, error) {
		attempts++
		return nil, apiErr("AccessDenied")
	}
	res := probeAccount(context.Background(), testAccount(), threeRegions, mk)

	if attempts != 1 {
		t.Errorf("attempted %d regions; an assume-role refusal is region-independent and should be asked once", attempts)
	}
	if res.Reachable {
		t.Error("want unreachable")
	}
	if !res.AssumeRoleDenied {
		t.Error("want AssumeRoleDenied so ApplyProbes can weigh it against this account's history")
	}
}

// Every region failing for a NON-denial reason is a plain failure, not a denial.
// The distinction matters: a denial gets the baseline check, a failure counts
// straight toward K.
func TestProbeAccount_TotalNonDenialFailureIsNotADenial(t *testing.T) {
	s := regionScript{credErr: map[string]error{}}
	for _, r := range threeRegions {
		s.credErr[r] = errors.New("sts timeout")
	}
	res := probeAccount(context.Background(), testAccount(), threeRegions, s.mk)
	if res.Reachable {
		t.Error("want unreachable")
	}
	if res.AssumeRoleDenied {
		t.Error("a timeout is not a refusal — mislabeling it would route it through the baseline check it does not deserve")
	}
}

// Credentials work everywhere but EC2 refuses in every region: that IS a denial
// (the role exists but lost its EC2 grant), and must be reported as one.
func TestProbeAccount_TotalEC2DenialIsADenial(t *testing.T) {
	s := regionScript{describeEr: map[string]error{}}
	for _, r := range threeRegions {
		s.describeEr[r] = apiErr("UnauthorizedOperation")
	}
	res := probeAccount(context.Background(), testAccount(), threeRegions, s.mk)
	if res.Reachable {
		t.Error("want unreachable")
	}
	if !res.AssumeRoleDenied {
		t.Error("EC2 refusing in every region is a refusal")
	}
}

// A mid-pagination error discards the partial count rather than returning it. A
// truncated count reads as "emptier than reality" — the direction that deprovisions.
// The fake succeeds on page 1 and fails on page 2 specifically so this can fail:
// with a first-page failure there is nothing accumulated to leak.
func TestCountManaged_PaginationErrorDiscardsPartialCount(t *testing.T) {
	got, err := countManaged(context.Background(), &pagedEC2{firstPage: 7})
	if err == nil {
		t.Fatal("expected the error to propagate")
	}
	if got != 0 {
		t.Errorf("count = %d, want 0 — a partial count must never be reported as a total", got)
	}
}

// And the same at the probe level: a mid-pagination failure makes the region's
// answer unusable, so coverage is partial and emptiness is unproven — the count
// from the page that did arrive must not be treated as that region's total.
func TestProbeAccount_MidPaginationFailureIsPartialCoverage(t *testing.T) {
	mk := func(ctx context.Context, acct accountlifecycle.Account, region string) (ec2API, error) {
		if region == "eu-west-1" {
			return &pagedEC2{firstPage: 7}, nil
		}
		return &fakeEC2{count: 0}, nil
	}
	res := probeAccount(context.Background(), testAccount(), threeRegions, mk)
	if !res.Reachable {
		t.Error("two regions answered fully, so the account is reachable")
	}
	if !res.EmptinessUnproven {
		t.Error("a region failed mid-pagination — its count is unknown, not zero")
	}
	if res.LiveInstances != 0 {
		t.Errorf("liveInstances = %d, want 0 — the partial page must not be counted", res.LiveInstances)
	}
}

// ── parseRegions ─────────────────────────────────────────────────────────────

// The region list is a correctness input: a region missing from it is a region
// whose instances are invisible to dormancy. An empty or blank override must fall
// back to the full default set, never to nothing.
func TestParseRegions(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{"unset falls back to defaults", "", defaultRegions},
		{"whitespace falls back to defaults", "   ", defaultRegions},
		{"commas only falls back to defaults", " , , ", defaultRegions},
		{"explicit list", "us-east-1,eu-west-1", []string{"us-east-1", "eu-west-1"}},
		{"trims spaces", " us-east-1 , eu-west-1 ", []string{"us-east-1", "eu-west-1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRegions(tc.env)
			if len(got) != len(tc.want) {
				t.Fatalf("parseRegions(%q) = %v, want %v", tc.env, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("parseRegions(%q)[%d] = %q, want %q", tc.env, i, got[i], tc.want[i])
				}
			}
		})
	}
}
