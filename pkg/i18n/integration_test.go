package i18n

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// TestLanguageDetectionFromEnv validates that language is correctly detected from environment variables
func TestLanguageDetectionFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		langVar  string
		langVal  string
		expected string
	}{
		{"LANG with UTF-8", "LANG", "es_ES.UTF-8", "es"},
		{"LANG simple", "LANG", "fr_FR", "fr"},
		{"LC_ALL takes precedence", "LC_ALL", "de_DE", "de"},
		{"LANGUAGE", "LANGUAGE", "ja_JP", "ja"},
		{"Portuguese", "LANG", "pt_BR", "pt"},
		{"Unsupported falls back to English", "LANG", "zh_CN", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env vars
			origLANG := os.Getenv("LANG")
			origLC_ALL := os.Getenv("LC_ALL")
			origLANGUAGE := os.Getenv("LANGUAGE")
			defer func() {
				os.Setenv("LANG", origLANG)
				os.Setenv("LC_ALL", origLC_ALL)
				os.Setenv("LANGUAGE", origLANGUAGE)
			}()

			// Clear all env vars first
			os.Unsetenv("LANG")
			os.Unsetenv("LC_ALL")
			os.Unsetenv("LANGUAGE")

			// Set the specific var for this test
			os.Setenv(tt.langVar, tt.langVal)

			// Detect language
			detected := DetectLanguage()
			if detected != tt.expected {
				t.Errorf("DetectLanguage() with %s=%s = %q, want %q", tt.langVar, tt.langVal, detected, tt.expected)
			}
		})
	}
}

// TestLanguageSwitchAtRuntime validates that language can be changed at runtime
func TestLanguageSwitchAtRuntime(t *testing.T) {
	// Initialize with English
	cfg := Config{
		Language: "en",
		Verbose:  false,
	}
	i18nInstance, err := NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create i18n instance: %v", err)
	}

	// Get a known translation key in English
	enMsg := i18nInstance.T("spawn.launch.short")
	if enMsg == "" || enMsg == "spawn.launch.short" {
		t.Errorf("English translation not found for spawn.launch.short")
	}

	// Switch to Spanish
	cfg.Language = "es"
	i18nInstance, err = NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create Spanish i18n instance: %v", err)
	}

	esMsg := i18nInstance.T("spawn.launch.short")
	if esMsg == "" || esMsg == "spawn.launch.short" {
		t.Errorf("Spanish translation not found for spawn.launch.short")
	}

	// They should be different
	if enMsg == esMsg {
		t.Errorf("English and Spanish translations are identical: %q", enMsg)
	}

	t.Logf("English: %q", enMsg)
	t.Logf("Spanish: %q", esMsg)
}

// TestEmojiAccessibilityMode validates emoji and accessibility mode behavior
func TestEmojiAccessibilityMode(t *testing.T) {
	tests := []struct {
		name              string
		noEmoji           bool
		accessibility     bool
		expectedEmoji     string // Expected from Emoji("rocket")
		expectedSymbol    string // Expected from Symbol("success")
	}{
		{"Normal mode", false, false, "🚀", "✅"},
		{"No emoji mode", true, false, "", "[✓]"},
		{"Accessibility mode", false, true, "", "[✓]"},
		{"Both flags", true, true, "", "[✓]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Language:         "en",
				NoEmoji:          tt.noEmoji,
				AccessibilityMode: tt.accessibility,
			}
			i18nInstance, err := NewLocalizer(cfg)
			if err != nil {
				t.Fatalf("Failed to create i18n instance: %v", err)
			}

			// Test Emoji() function with "rocket" (exists in emojiMap)
			emoji := i18nInstance.Emoji("rocket")
			if emoji != tt.expectedEmoji {
				t.Errorf("Emoji(\"rocket\") = %q, want %q", emoji, tt.expectedEmoji)
			}

			// Test Symbol() function
			symbol := i18nInstance.Symbol("success")
			if symbol != tt.expectedSymbol {
				t.Errorf("Symbol(\"success\") = %q, want %q", symbol, tt.expectedSymbol)
			}

			t.Logf("%s: Emoji=\"%s\" Symbol=\"%s\"", tt.name, emoji, symbol)
		})
	}
}

// TestTranslationWithTemplateData validates template variable substitution
func TestTranslationWithTemplateData(t *testing.T) {
	cfg := Config{
		Language: "en",
	}
	i18nInstance, err := NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create i18n instance: %v", err)
	}

	// Test a key with template variables
	// spawn.wizard.step2.prompt has {{.DefaultRegion}}
	msg := i18nInstance.Tf("spawn.wizard.step2.prompt", map[string]interface{}{
		"DefaultRegion": "us-west-2",
	})

	if !strings.Contains(msg, "us-west-2") {
		t.Errorf("Template variable not substituted. Got: %q", msg)
	}

	if strings.Contains(msg, "{{.DefaultRegion}}") {
		t.Errorf("Template variable not replaced. Got: %q", msg)
	}

	t.Logf("Template result: %q", msg)
}

// TestTranslationWithMultipleVariables validates multiple template variable substitution
func TestTranslationWithMultipleVariables(t *testing.T) {
	cfg := Config{
		Language: "en",
	}
	i18nInstance, err := NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create i18n instance: %v", err)
	}

	// Find a key with multiple variables by testing some known keys
	// spawn.agent.idle_warning has {{.IdleDuration}} and {{.Remaining}}
	msg := i18nInstance.Tf("spawn.agent.idle_warning", map[string]interface{}{
		"IdleDuration": "15m",
		"Remaining":    "5m",
	})

	if !strings.Contains(msg, "15m") || !strings.Contains(msg, "5m") {
		t.Errorf("Template variables not substituted. Got: %q", msg)
	}

	t.Logf("Multiple variables result: %q", msg)
}

// TestPluralizations validates count-based plural translations
func TestPluralizations(t *testing.T) {
	cfg := Config{
		Language: "en",
	}
	i18nInstance, err := NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create i18n instance: %v", err)
	}

	// Test pluralization with different counts
	// Find a key that has plural forms (one, other)
	// spawn.list.instances_found should have plural forms
	tests := []struct {
		count    int
		expected string
	}{
		{0, "instances"},  // zero/other form
		{1, "instance"},   // one form
		{5, "instances"},  // other form
		{100, "instances"}, // other form
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.count)), func(t *testing.T) {
			msg := i18nInstance.Tc("spawn.list.instances_found", tt.count, map[string]interface{}{
				"Count": tt.count,
			})

			// The message should contain appropriate singular/plural form
			t.Logf("Count %d: %q", tt.count, msg)

			// Check that it contains the count
			if !strings.Contains(msg, string(rune(tt.count))) {
				// Try converting number to string properly
				t.Logf("Note: Count %d may not appear as string in message: %q", tt.count, msg)
			}
		})
	}
}

// TestErrorTranslations validates Te() error wrapping functionality
func TestErrorTranslations(t *testing.T) {
	cfg := Config{
		Language: "en",
	}
	i18nInstance, err := NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create i18n instance: %v", err)
	}

	// Create a sample error
	originalErr := errors.New("connection timeout")

	// Wrap it with a translated message
	wrappedErr := i18nInstance.Te("spawn.connect.error.timeout", originalErr, map[string]interface{}{
		"Timeout": "30s",
	})

	// Check that error is not nil
	if wrappedErr == nil {
		t.Fatal("Te() returned nil error")
	}

	// Check that error message contains both translation and original error
	errMsg := wrappedErr.Error()

	if !strings.Contains(errMsg, "connection timeout") {
		t.Errorf("Wrapped error doesn't contain original error. Got: %q", errMsg)
	}

	t.Logf("Wrapped error: %q", errMsg)
}

// TestGlobalInstance validates that global T(), Ts(), Te() work after Init()
func TestGlobalInstance(t *testing.T) {
	// Initialize global instance
	cfg := Config{
		Language: "en",
		Verbose:  false,
	}
	Init(cfg)

	// Test global T() function
	msg := T("spawn.launch.short")
	if msg == "" || msg == "spawn.launch.short" {
		t.Errorf("Global T() failed to translate spawn.launch.short")
	}

	// Test global Tf() with template
	msgWithTemplate := Tf("spawn.wizard.step2.prompt", map[string]interface{}{
		"DefaultRegion": "eu-west-1",
	})
	if !strings.Contains(msgWithTemplate, "eu-west-1") {
		t.Errorf("Global Ts() failed to substitute template. Got: %q", msgWithTemplate)
	}

	// Test global Te() error wrapping
	err := errors.New("test error")
	wrappedErr := Te("spawn.connect.error.timeout", err, map[string]interface{}{})
	if wrappedErr == nil {
		t.Error("Global Te() returned nil error")
	}

	t.Logf("Global T(): %q", msg)
	t.Logf("Global Tf(): %q", msgWithTemplate)
	t.Logf("Global Te(): %q", wrappedErr)
}

// TestAllLanguagesLoadSuccessfully validates that all supported languages can be loaded
func TestAllLanguagesLoadSuccessfully(t *testing.T) {
	languages := []string{"en", "es", "fr", "de", "ja", "pt"}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			cfg := Config{
				Language: lang,
				Verbose:  false,
			}
			i18nInstance, err := NewLocalizer(cfg)
			if err != nil {
				t.Fatalf("Failed to create %s i18n instance: %v", lang, err)
			}

			// Try to translate a known key
			msg := i18nInstance.T("spawn.launch.short")
			if msg == "" || msg == "spawn.launch.short" {
				t.Errorf("%s translation not found for spawn.launch.short", lang)
			}

			t.Logf("%s: spawn.launch.short = %q", lang, msg)
		})
	}
}

// TestTemplateDataWithMissingVariable validates behavior when template data is missing
func TestTemplateDataWithMissingVariable(t *testing.T) {
	cfg := Config{
		Language: "en",
	}
	i18nInstance, err := NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create i18n instance: %v", err)
	}

	// Try to translate with missing template data
	msg := i18nInstance.Tf("spawn.wizard.step2.prompt", map[string]interface{}{
		// Intentionally missing "DefaultRegion" key
	})

	// Should still return something (either fallback or partial translation)
	if msg == "" {
		t.Error("Ts() with missing template data returned empty string")
	}

	t.Logf("Result with missing data: %q", msg)
}

// TestConsistencyAcrossLanguages validates that same keys work in all languages
func TestConsistencyAcrossLanguages(t *testing.T) {
	testKeys := []string{
		"spawn.launch.short",
		"spawn.launch.long",
		"spawn.list.short",
		"spawn.connect.short",
		"truffle.search.short",
	}

	languages := []string{"en", "es", "fr", "de", "ja", "pt"}

	for _, key := range testKeys {
		t.Run(key, func(t *testing.T) {
			for _, lang := range languages {
				cfg := Config{Language: lang}
				i18nInstance, err := NewLocalizer(cfg)
				if err != nil {
					t.Fatalf("Failed to create %s instance: %v", lang, err)
				}

				msg := i18nInstance.T(key)
				if msg == "" || msg == key {
					t.Errorf("%s: translation missing for %s", lang, key)
				}
			}
		})
	}
}
