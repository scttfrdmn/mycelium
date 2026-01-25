package security

import (
	"testing"
)

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expected string
	}{
		{
			name:     "empty string",
			secret:   "",
			expected: "",
		},
		{
			name:     "short secret",
			secret:   "abc123",
			expected: "****",
		},
		{
			name:     "exactly 8 chars",
			secret:   "12345678",
			expected: "****",
		},
		{
			name:     "long secret",
			secret:   "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX",
			expected: "http****XXXX",
		},
		{
			name:     "api key",
			secret:   "sk-1234567890abcdef1234567890abcdef",
			expected: "sk-1****cdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSecret(tt.secret)
			if result != tt.expected {
				t.Errorf("MaskSecret() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "empty string",
			value:    "",
			expected: false,
		},
		{
			name:     "plaintext URL",
			value:    "https://hooks.slack.com/services/T00/B00/XXX",
			expected: false,
		},
		{
			name:     "short base64",
			value:    "SGVsbG8gV29ybGQ=",
			expected: false,
		},
		{
			name:     "long base64 (simulated encrypted)",
			value:    "AQICAHhPkQqJxqH0TdKlPqVoGMeXmVvjJdQkWqPYKzNxQqRzGwF8kLmNoPqRsTuVwXyZaBcDAAAAfjB8BgkqhkiG9w0BBwagbzBtAgEAMGgGCSqGSIb3DQEHATAeBglghkgBZQMEAS4wEQQMJKxLmNoPqRsTuVwXAgEQgDsxQqRzGwF8kLmNoPqRsTuVwXyZaBcD",
			expected: true,
		},
		{
			name:     "not base64",
			value:    "this is not base64!",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEncrypted(tt.value)
			if result != tt.expected {
				t.Errorf("IsEncrypted() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "slack webhook",
			url:      "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX",
			expected: "https://hooks.slack.com/****",
		},
		{
			name:     "generic webhook",
			url:      "https://example.com/webhook/secret/path",
			expected: "https://example.com/****",
		},
		{
			name:     "URL without path",
			url:      "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "not a URL",
			url:      "not-a-url",
			expected: "not-****-url",
		},
		{
			name:     "email (not URL)",
			url:      "user@example.com",
			expected: "user****.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskURL(tt.url)
			if result != tt.expected {
				t.Errorf("MaskURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Note: EncryptSecret and DecryptSecret require actual AWS KMS client
// and would need integration tests with proper AWS credentials.
// Unit tests would need mocking, which is beyond the scope here.
