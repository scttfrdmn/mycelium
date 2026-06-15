package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// mkJWT builds a signed RS256 JWT for testing. kid selects which key the
// verifier will look up; claims is marshaled as the payload.
func mkJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	hdr := map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid}
	hb, _ := json.Marshal(hdr)
	cb, _ := json.Marshal(claims)
	seg := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(cb)
	h := sha256.Sum256([]byte(seg))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return seg + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// withTestKey injects a public key into the JWKS cache under kid and marks it
// fresh, then restores the prior cache state on cleanup.
func withTestKey(t *testing.T, kid string, pub *rsa.PublicKey) {
	t.Helper()
	botFrameworkKeys.mu.Lock()
	prevKeys, prevAt := botFrameworkKeys.keys, botFrameworkKeys.fetchedAt
	botFrameworkKeys.keys = map[string]*rsa.PublicKey{kid: pub}
	botFrameworkKeys.fetchedAt = time.Now()
	botFrameworkKeys.mu.Unlock()
	t.Cleanup(func() {
		botFrameworkKeys.mu.Lock()
		botFrameworkKeys.keys, botFrameworkKeys.fetchedAt = prevKeys, prevAt
		botFrameworkKeys.mu.Unlock()
	})
}

func validClaims(appID string) map[string]any {
	return map[string]any{
		"iss": botFrameworkIssuer,
		"aud": appID,
		"exp": time.Now().Add(time.Hour).Unix(),
		"nbf": time.Now().Add(-time.Minute).Unix(),
	}
}

func TestVerifyTeamsJWT_Valid(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	withTestKey(t, "k1", &key.PublicKey)
	tok := mkJWT(t, key, "k1", validClaims("bot-app-id"))
	if err := verifyTeamsJWT(context.Background(), "Bearer "+tok, "bot-app-id"); err != nil {
		t.Errorf("expected valid token to pass, got: %v", err)
	}
}

func TestVerifyTeamsJWT_NoAppID(t *testing.T) {
	if err := verifyTeamsJWT(context.Background(), "Bearer x", ""); err == nil {
		t.Error("expected missing App ID to fail")
	}
}

func TestVerifyTeamsJWT_WrongAudience(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	withTestKey(t, "k1", &key.PublicKey)
	tok := mkJWT(t, key, "k1", validClaims("some-other-bot"))
	if err := verifyTeamsJWT(context.Background(), "Bearer "+tok, "bot-app-id"); err == nil {
		t.Error("expected wrong audience to fail")
	}
}

func TestVerifyTeamsJWT_WrongIssuer(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	withTestKey(t, "k1", &key.PublicKey)
	claims := validClaims("bot-app-id")
	claims["iss"] = "https://evil.example.com"
	tok := mkJWT(t, key, "k1", claims)
	if err := verifyTeamsJWT(context.Background(), "Bearer "+tok, "bot-app-id"); err == nil {
		t.Error("expected wrong issuer to fail")
	}
}

func TestVerifyTeamsJWT_Expired(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	withTestKey(t, "k1", &key.PublicKey)
	claims := validClaims("bot-app-id")
	claims["exp"] = time.Now().Add(-time.Hour).Unix()
	tok := mkJWT(t, key, "k1", claims)
	if err := verifyTeamsJWT(context.Background(), "Bearer "+tok, "bot-app-id"); err == nil {
		t.Error("expected expired token to fail")
	}
}

func TestVerifyTeamsJWT_WrongKey(t *testing.T) {
	signKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	withTestKey(t, "k1", &otherKey.PublicKey) // cache holds a DIFFERENT key for k1
	tok := mkJWT(t, signKey, "k1", validClaims("bot-app-id"))
	if err := verifyTeamsJWT(context.Background(), "Bearer "+tok, "bot-app-id"); err == nil {
		t.Error("expected signature mismatch to fail")
	}
}

func TestVerifyTeamsJWT_AlgNone(t *testing.T) {
	// "alg":"none" with an empty signature must be rejected outright.
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","kid":"k1"}`))
	cb, _ := json.Marshal(validClaims("bot-app-id"))
	claims := base64.RawURLEncoding.EncodeToString(cb)
	tok := hdr + "." + claims + "."
	if err := verifyTeamsJWT(context.Background(), "Bearer "+tok, "bot-app-id"); err == nil {
		t.Error("expected alg=none to fail")
	}
}

func TestVerifyTeamsJWT_Malformed(t *testing.T) {
	if err := verifyTeamsJWT(context.Background(), "Bearer not-a-jwt", "bot-app-id"); err == nil {
		t.Error("expected malformed token to fail")
	}
}

func TestVerifyTeamsJWT_AudienceArray(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	withTestKey(t, "k1", &key.PublicKey)
	claims := validClaims("bot-app-id")
	claims["aud"] = []string{"other", "bot-app-id"}
	tok := mkJWT(t, key, "k1", claims)
	if err := verifyTeamsJWT(context.Background(), "Bearer "+tok, "bot-app-id"); err != nil {
		t.Errorf("expected audience array containing App ID to pass, got: %v", err)
	}
}
