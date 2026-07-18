package main

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// TestGenerateExternalID covers the per-registration STS ExternalId (#374):
// it must be high-entropy, unique, and carry the self-describing prefix.
func TestGenerateExternalID(t *testing.T) {
	a, err := generateExternalID()
	if err != nil {
		t.Fatalf("generateExternalID: %v", err)
	}
	if !strings.HasPrefix(a, "spore-") {
		t.Errorf("external id %q missing spore- prefix", a)
	}
	// "spore-" + 64 hex chars (32 bytes of entropy).
	if len(a) != len("spore-")+64 {
		t.Errorf("external id len = %d, want %d", len(a), len("spore-")+64)
	}
	b, _ := generateExternalID()
	if a == b {
		t.Error("two generated external ids collided")
	}
	// It must never equal the (now-removed) shared static value.
	if a == "spawn-bot" {
		t.Error("generated external id equals the removed static value")
	}
}

// TestCrossAccountEC2_FailsClosedWithoutExternalID covers #413: a registration
// with no per-account ExternalId can no longer assume its role via the removed
// shared "spawn-bot" fallback — it must fail closed.
func TestCrossAccountEC2_FailsClosedWithoutExternalID(t *testing.T) {
	// A non-empty BOT_EXTERNAL_ID env must NOT resurrect the old fallback.
	t.Setenv("BOT_EXTERNAL_ID", "spawn-bot")
	_, err := crossAccountEC2(context.Background(), aws.Config{},
		"arn:aws:iam::123456789012:role/SpawnBotCrossAccount", "i-0abc", "")
	if err == nil {
		t.Fatal("expected error when externalID is empty, got nil (fallback not removed?)")
	}
	if !strings.Contains(err.Error(), "external_id") {
		t.Errorf("error %q should mention the missing external_id", err)
	}
}
