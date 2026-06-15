package main

import (
	"strings"
	"testing"
)

func TestOAuthSecret_FailClosed(t *testing.T) {
	cases := []struct {
		name   string
		val    string
		setEnv bool
		wantOK bool
	}{
		{"unset", "", false, false},
		{"empty", "", true, false},
		{"placeholder", "change-me", true, false},
		{"real", "s3cr3t-value", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setEnv {
				t.Setenv("BOT_OAUTH_SECRET", tc.val)
			} else {
				t.Setenv("BOT_OAUTH_SECRET", "") // ensure clean
			}
			_, err := oauthSecret()
			if (err == nil) != tc.wantOK {
				t.Errorf("oauthSecret() err=%v, wantOK=%v", err, tc.wantOK)
			}
		})
	}
}

func TestOAuthSignedState_RoundTrip(t *testing.T) {
	t.Setenv("BOT_OAUTH_SECRET", "round-trip-secret")
	state, err := oauthSignedState("slack", "verifier-123")
	if err != nil {
		t.Fatalf("oauthSignedState: %v", err)
	}
	got, err := oauthExtractVerifier("slack", state)
	if err != nil {
		t.Fatalf("oauthExtractVerifier: %v", err)
	}
	if got != "verifier-123" {
		t.Errorf("verifier = %q, want verifier-123", got)
	}
}

func TestOAuthExtractVerifier_ForgedWithPlaceholder(t *testing.T) {
	// A state forged with the old "change-me" default must not validate once a
	// real secret is configured.
	t.Setenv("BOT_OAUTH_SECRET", "change-me")
	forged, err := oauthSignedState("slack", "attacker")
	if err == nil {
		// With the fix, signing under "change-me" itself fails closed.
		t.Fatalf("expected signing under placeholder to fail, got state %q", forged)
	}
}

func TestGenerateConnectCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		c, err := generateConnectCode()
		if err != nil {
			t.Fatalf("generateConnectCode: %v", err)
		}
		if len(c) != 8 {
			t.Errorf("code %q length = %d, want 8", c, len(c))
		}
		if c != strings.ToUpper(c) {
			t.Errorf("code %q not uppercase", c)
		}
		if seen[c] {
			t.Errorf("duplicate code %q within 100 draws", c)
		}
		seen[c] = true
	}
}
