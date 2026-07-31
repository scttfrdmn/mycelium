package main

import (
	"testing"
)

const verified = "123456789012"

// The core security invariant: validate trusts ONLY the SigV4-verified account,
// and rejects any roleArn that doesn't live in it — no AWS needed.
func TestValidate_SecurityInvariant(t *testing.T) {
	tests := []struct {
		name       string
		req        phoneHomeRequest
		wantStatus int // 0 = accept
	}{
		{
			name:       "role in the verified account is accepted",
			req:        phoneHomeRequest{RoleArn: "arn:aws:iam::123456789012:role/spore-portal-onboard", ExternalId: "abc123", Region: "us-east-1"},
			wantStatus: 0,
		},
		{
			name:       "role in a DIFFERENT account is rejected (spoof attempt)",
			req:        phoneHomeRequest{RoleArn: "arn:aws:iam::999999999999:role/evil", ExternalId: "abc123", Region: "us-east-1"},
			wantStatus: 403,
		},
		{
			name:       "malformed roleArn is rejected",
			req:        phoneHomeRequest{RoleArn: "not-an-arn", ExternalId: "abc123"},
			wantStatus: 400,
		},
		{
			name:       "missing externalId is rejected",
			req:        phoneHomeRequest{RoleArn: "arn:aws:iam::123456789012:role/x", ExternalId: ""},
			wantStatus: 400,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			acct, verr := validate(verified, "arn:aws:iam::123456789012:user/caller", tc.req)
			if tc.wantStatus == 0 {
				if verr != nil {
					t.Fatalf("expected accept, got %d: %s", verr.status, verr.msg)
				}
				if acct.AccountID != verified {
					t.Errorf("account id = %q, want %q (must come from the verified caller, not the body)", acct.AccountID, verified)
				}
				return
			}
			if verr == nil {
				t.Fatalf("expected reject %d, got accept", tc.wantStatus)
			}
			if verr.status != tc.wantStatus {
				t.Errorf("status = %d, want %d", verr.status, tc.wantStatus)
			}
		})
	}
}

// validate defaults an empty region to us-east-1.
func TestValidate_DefaultRegion(t *testing.T) {
	acct, verr := validate(verified, "caller", phoneHomeRequest{
		RoleArn: "arn:aws:iam::123456789012:role/x", ExternalId: "e",
	})
	if verr != nil {
		t.Fatalf("unexpected reject: %s", verr.msg)
	}
	if acct.Region != "us-east-1" {
		t.Errorf("region = %q, want us-east-1", acct.Region)
	}
}
