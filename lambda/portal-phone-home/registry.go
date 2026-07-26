package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Account is one onboarded BYOA account's registration. AccountID is the
// partition key. Everything here is derived from the SigV4-verified caller
// except the ExternalId + Region the caller supplies.
type Account struct {
	AccountID    string `json:"accountId" dynamodbav:"accountId"`
	RoleArn      string `json:"roleArn" dynamodbav:"roleArn"`
	ExternalId   string `json:"externalId" dynamodbav:"externalId"`
	Region       string `json:"region" dynamodbav:"region"`
	RegisteredBy string `json:"registeredBy" dynamodbav:"registeredBy"`
	RegisteredAt string `json:"registeredAt" dynamodbav:"registeredAt"`
}

type Registry struct {
	client       *dynamodb.Client
	accountTable string
}

func newRegistry(cfg aws.Config) *Registry {
	table := os.Getenv("ACCOUNTS_TABLE")
	if table == "" {
		table = "spore-portal-accounts"
	}
	return &Registry{
		client:       dynamodb.NewFromConfig(cfg),
		accountTable: table,
	}
}

// PutAccount upserts a registration. Re-onboarding the same account overwrites
// the prior row (a fresh ExternalId/role supersedes the old one).
func (r *Registry) PutAccount(ctx context.Context, acct *Account) error {
	if acct.RegisteredAt == "" {
		acct.RegisteredAt = time.Now().UTC().Format(time.RFC3339)
	}
	item, err := attributevalue.MarshalMap(acct)
	if err != nil {
		return fmt.Errorf("marshal account: %w", err)
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.accountTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put item: %w", err)
	}
	return nil
}

// GetAccount fetches a registration by account id (used by the portal/tests).
func (r *Registry) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.accountTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"accountId": &dynamodbtypes.AttributeValueMemberS{Value: accountID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	if out.Item == nil {
		return nil, nil
	}
	var acct Account
	if err := attributevalue.UnmarshalMap(out.Item, &acct); err != nil {
		return nil, fmt.Errorf("unmarshal account: %w", err)
	}
	return &acct, nil
}
