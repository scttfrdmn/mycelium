package main

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	substrate "github.com/scttfrdmn/substrate"
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

// Round-trip PutAccount → GetAccount against substrate's in-memory DynamoDB.
func TestRegistry_PutGet(t *testing.T) {
	r := newTestRegistry(t)
	ctx := context.Background()

	acct := &Account{
		AccountID:    verified,
		RoleArn:      "arn:aws:iam::123456789012:role/spore-portal-onboard",
		ExternalId:   "high-entropy-external-id",
		Region:       "us-west-2",
		RegisteredBy: "arn:aws:iam::123456789012:user/alice",
	}
	if err := r.PutAccount(ctx, acct); err != nil {
		t.Fatalf("PutAccount: %v", err)
	}
	got, err := r.GetAccount(ctx, verified)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got == nil {
		t.Fatal("GetAccount returned nil for a just-registered account")
	}
	if got.RoleArn != acct.RoleArn || got.ExternalId != acct.ExternalId || got.Region != acct.Region {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, acct)
	}
	if got.RegisteredAt == "" {
		t.Error("RegisteredAt should be stamped on Put")
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	r := newTestRegistry(t)
	got, err := r.GetAccount(context.Background(), "000000000000")
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for an unregistered account, got %+v", got)
	}
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	ts := substrate.StartTestServer(t)
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		awsconfig.WithBaseEndpoint(ts.URL),
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	client := dynamodb.NewFromConfig(cfg)
	createAccountsTable(t, client)
	return &Registry{client: client, accountTable: "spore-portal-accounts"}
}

func createAccountsTable(t *testing.T, client *dynamodb.Client) {
	t.Helper()
	_, err := client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName:   aws.String("spore-portal-accounts"),
		BillingMode: dynamodbtypes.BillingModePayPerRequest,
		AttributeDefinitions: []dynamodbtypes.AttributeDefinition{
			{AttributeName: aws.String("accountId"), AttributeType: dynamodbtypes.ScalarAttributeTypeS},
		},
		KeySchema: []dynamodbtypes.KeySchemaElement{
			{AttributeName: aws.String("accountId"), KeyType: dynamodbtypes.KeyTypeHash},
		},
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
}
