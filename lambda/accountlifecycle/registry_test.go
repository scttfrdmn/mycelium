package accountlifecycle

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	substrate "github.com/scttfrdmn/substrate/emulator"
)

const verified = "123456789012"

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
