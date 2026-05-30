package main

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestParseJSON(t *testing.T) {
	var v struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	if err := parseJSON(`{"name":"x","n":3}`, &v); err != nil {
		t.Fatalf("parseJSON: %v", err)
	}
	if v.Name != "x" || v.N != 3 {
		t.Errorf("parsed %+v, want {x 3}", v)
	}
	if err := parseJSON(`{not json`, &v); err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestTrimAndSplit(t *testing.T) {
	if trim("  hi  ") != "hi" {
		t.Error("trim failed")
	}
	if got := splitString("a,b,c", ","); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("splitString = %v", got)
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"us-east-1,us-west-2", []string{"us-east-1", "us-west-2"}},
		{" a , b ,, c ", []string{"a", "b", "c"}}, // trims + drops empties
		{"", []string{}},
		{",,", []string{}},
	}
	for _, tt := range tests {
		got := splitCSV(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestErrResp(t *testing.T) {
	resp := errResp(http.StatusNotFound, "nope")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("missing JSON content-type header")
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["error"] != "nope" {
		t.Errorf("error body = %q, want nope", body["error"])
	}
}

func TestJSONResp(t *testing.T) {
	resp := jsonResp(http.StatusOK, map[string]int{"count": 5})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(resp.Body, `"count":5`) {
		t.Errorf("body = %q, want count:5", resp.Body)
	}
}

func TestGenerateAPIKey(t *testing.T) {
	k1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if !strings.HasPrefix(k1, "sk_") {
		t.Errorf("key %q missing sk_ prefix", k1)
	}
	// sk_ + 48 hex chars (24 bytes).
	if len(k1) != 3+48 {
		t.Errorf("key length = %d, want 51", len(k1))
	}
	// Keys must be unique.
	k2, _ := GenerateAPIKey()
	if k1 == k2 {
		t.Error("two generated keys collided")
	}
}

func TestDecodeOptions(t *testing.T) {
	got := decodeOptions("ttl=4h,idle=30m,bad,name=test")
	want := map[string]string{"ttl": "4h", "idle": "30m", "name": "test"} // "bad" (no =) skipped
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decodeOptions = %v, want %v", got, want)
	}
	if len(decodeOptions("")) != 0 {
		t.Error("decodeOptions(\"\") should be empty")
	}
}

func TestBuildOptionsHint(t *testing.T) {
	// Keys are sorted for stable output.
	got := buildOptionsHint(map[string]string{"ttl": "4h", "idle": "30m", "name": "x"})
	if got != "idle, name, ttl" {
		t.Errorf("buildOptionsHint = %q, want 'idle, name, ttl'", got)
	}
}

// --- handler routing (auth happens before any AWS call) ---

func TestHandler_MissingAPIKey(t *testing.T) {
	req := events.APIGatewayV2HTTPRequest{}
	req.RequestContext.HTTP.Method = "GET"
	req.RequestContext.HTTP.Path = "/v1/instances"

	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler transport error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for missing API key", resp.StatusCode)
	}
}
