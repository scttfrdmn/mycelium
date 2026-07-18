package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Principal represents a validated API key caller.
type Principal struct {
	KeyID     string
	Project   string // "spore", "prism", etc.
	AccountID string // AWS account the key belongs to
	CreatedAt time.Time
}

// hashKey returns the hex-encoded SHA-256 of an API key. Keys are stored and
// looked up by this hash so a table dump never yields usable credentials
// (#374). The hash is also the source of the truncated KeyID we log.
func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// keyID returns a short, non-secret identifier for a key, safe to log. It is
// the first 8 hex chars of the key's SHA-256 — stable per key, but not
// reversible to the secret (unlike the old key[:8], which leaked the prefix).
func keyID(key string) string {
	return hashKey(key)[:8]
}

func validateAPIKey(ctx context.Context, key string) (*Principal, error) {
	table := os.Getenv("API_KEYS_TABLE")
	if table == "" {
		table = "spore-api-keys"
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(cfg)

	// Dual-read migration (#374): keys are stored hashed, but legacy rows still
	// key on the plaintext. Look up by hash first; on a miss, fall back to the
	// plaintext PK and, on a hit, rewrite the row to its hashed form so the
	// plaintext copy disappears over time. New keys are only ever written hashed.
	hashed := hashKey(key)
	item, err := getKeyItem(ctx, client, table, hashed)
	if err != nil {
		return nil, err
	}
	if item == nil {
		item, err = getKeyItem(ctx, client, table, key)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, fmt.Errorf("key not found")
		}
		// Best-effort migrate legacy plaintext row → hashed. A failure here must
		// not fail the request; the row is re-migrated on the next call.
		migrateKeyToHash(ctx, client, table, key, hashed, item)
	}

	get := func(k string) string {
		if v, ok := item[k].(*dynamodbtypes.AttributeValueMemberS); ok {
			return v.Value
		}
		return ""
	}

	// Check revoked
	if get("revoked") == "true" {
		return nil, fmt.Errorf("key revoked")
	}

	return &Principal{
		KeyID:     keyID(key),
		Project:   get("project"),
		AccountID: get("account_id"),
	}, nil
}

// getKeyItem fetches the row whose api_key partition key equals pk (a hash or,
// for legacy rows, the plaintext key). Returns (nil, nil) when absent.
func getKeyItem(ctx context.Context, client *dynamodb.Client, table, pk string) (map[string]dynamodbtypes.AttributeValue, error) {
	out, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key: map[string]dynamodbtypes.AttributeValue{
			"api_key": &dynamodbtypes.AttributeValueMemberS{Value: pk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("lookup api key: %w", err)
	}
	return out.Item, nil
}

// migrateKeyToHash rewrites a legacy plaintext-keyed row under its hashed PK and
// deletes the plaintext row. Best-effort: any error is logged and swallowed so a
// migration hiccup never blocks an otherwise-valid caller. The write is
// conditioned on the hashed row not already existing to avoid clobbering.
func migrateKeyToHash(ctx context.Context, client *dynamodb.Client, table, plaintext, hashed string, item map[string]dynamodbtypes.AttributeValue) {
	// Guard against a corrupt row whose stored api_key isn't the plaintext we
	// looked up — hashing that would strand it under the wrong PK.
	if v, ok := item["api_key"].(*dynamodbtypes.AttributeValueMemberS); !ok || subtle.ConstantTimeCompare([]byte(v.Value), []byte(plaintext)) != 1 {
		return
	}
	newItem := make(map[string]dynamodbtypes.AttributeValue, len(item))
	for k, v := range item {
		newItem[k] = v
	}
	newItem["api_key"] = &dynamodbtypes.AttributeValueMemberS{Value: hashed}
	if _, err := client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(table),
		Item:                newItem,
		ConditionExpression: aws.String("attribute_not_exists(api_key)"),
	}); err != nil {
		// ConditionalCheckFailed means the hashed row already exists (a
		// concurrent migration) — fine; still drop the plaintext copy below.
		if !isConditionalCheckFailed(err) {
			log.Printf("api-key migrate: put hashed row failed for keyid=%s: %v", keyID(plaintext), err)
			return
		}
	}
	if _, err := client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(table),
		Key: map[string]dynamodbtypes.AttributeValue{
			"api_key": &dynamodbtypes.AttributeValueMemberS{Value: plaintext},
		},
	}); err != nil {
		log.Printf("api-key migrate: delete plaintext row failed for keyid=%s: %v", keyID(plaintext), err)
	}
}

func isConditionalCheckFailed(err error) bool {
	var cf *dynamodbtypes.ConditionalCheckFailedException
	return errors.As(err, &cf)
}

// GenerateAPIKey creates a new random API key (used by spawn api-key create).
func GenerateAPIKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk_" + hex.EncodeToString(b), nil
}
