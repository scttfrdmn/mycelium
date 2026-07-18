package main

import (
	"strings"
	"testing"
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
	// It must never equal the shared static fallback.
	if a == "spawn-bot" {
		t.Error("generated external id equals the static fallback")
	}
}
