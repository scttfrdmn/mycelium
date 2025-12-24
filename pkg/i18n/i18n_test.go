package i18n

import (
	"os"
	"testing"
)

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"English US", "en_US", "en"},
		{"English US UTF-8", "en_US.UTF-8", "en"},
		{"English UK", "en-GB", "en"},
		{"Spanish Spain", "es_ES", "es"},
		{"Spanish Mexico", "es-MX", "es"},
		{"French France", "fr_FR", "fr"},
		{"French Canada", "fr-CA", "fr"},
		{"German Germany", "de_DE", "de"},
		{"German Austria", "de-AT", "de"},
		{"Japanese", "ja_JP", "ja"},
		{"Japanese UTF-8", "ja_JP.UTF-8", "ja"},
		{"Uppercase", "EN_US", "en"},
		{"Already normalized", "en", "en"},
		{"Portuguese Brazil", "pt_BR", "pt"}, // Portuguese now supported
		{"Unsupported", "zh_CN", "en"},        // Chinese falls back to English
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeLanguage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectFromSystem(t *testing.T) {
	// Save original env vars
	origLANG := os.Getenv("LANG")
	origLC_ALL := os.Getenv("LC_ALL")
	origLANGUAGE := os.Getenv("LANGUAGE")

	// Restore after test
	defer func() {
		os.Setenv("LANG", origLANG)
		os.Setenv("LC_ALL", origLC_ALL)
		os.Setenv("LANGUAGE", origLANGUAGE)
	}()

	tests := []struct {
		name         string
		envLANG      string
		envLC_ALL    string
		envLANGUAGE  string
		expectedLang string
		expectedOk   bool
	}{
		{
			name:         "LANG set",
			envLANG:      "es_ES.UTF-8",
			envLC_ALL:    "",
			envLANGUAGE:  "",
			expectedLang: "es_ES.UTF-8",
			expectedOk:   true,
		},
		{
			name:         "LC_ALL set",
			envLANG:      "",
			envLC_ALL:    "fr_FR.UTF-8",
			envLANGUAGE:  "",
			expectedLang: "fr_FR.UTF-8",
			expectedOk:   true,
		},
		{
			name:         "LANGUAGE set",
			envLANG:      "",
			envLC_ALL:    "",
			envLANGUAGE:  "de:en",
			expectedLang: "de",
			expectedOk:   true,
		},
		{
			name:         "None set",
			envLANG:      "",
			envLC_ALL:    "",
			envLANGUAGE:  "",
			expectedLang: "",
			expectedOk:   false,
		},
		{
			name:         "LANG takes priority",
			envLANG:      "es_ES.UTF-8",
			envLC_ALL:    "fr_FR.UTF-8",
			envLANGUAGE:  "de:en",
			expectedLang: "es_ES.UTF-8",
			expectedOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test env vars
			os.Setenv("LANG", tt.envLANG)
			os.Setenv("LC_ALL", tt.envLC_ALL)
			os.Setenv("LANGUAGE", tt.envLANGUAGE)

			lang, ok := DetectFromSystem()
			if ok != tt.expectedOk {
				t.Errorf("DetectFromSystem() ok = %v, want %v", ok, tt.expectedOk)
			}
			if lang != tt.expectedLang {
				t.Errorf("DetectFromSystem() lang = %q, want %q", lang, tt.expectedLang)
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		lang     string
		expected bool
	}{
		{"en", true},
		{"es", true},
		{"fr", true},
		{"de", true},
		{"ja", true},
		{"pt", true},
		{"zh", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := IsSupported(tt.lang)
			if result != tt.expected {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestLocalizerCreation(t *testing.T) {
	cfg := Config{
		Language: "en",
		Verbose:  false,
	}

	localizer, err := NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("NewLocalizer() error = %v", err)
	}

	if localizer.Language() != "en" {
		t.Errorf("Language() = %q, want %q", localizer.Language(), "en")
	}

	if localizer.AccessibilityMode() {
		t.Error("AccessibilityMode() = true, want false")
	}

	if localizer.NoEmoji() {
		t.Error("NoEmoji() = true, want false")
	}
}

func TestAccessibilityModeImpliesNoEmoji(t *testing.T) {
	cfg := Config{
		Language:          "en",
		AccessibilityMode: true,
		NoEmoji:           false, // Should be overridden
	}

	localizer, err := NewLocalizer(cfg)
	if err != nil {
		t.Fatalf("NewLocalizer() error = %v", err)
	}

	if !localizer.AccessibilityMode() {
		t.Error("AccessibilityMode() = false, want true")
	}

	if !localizer.NoEmoji() {
		t.Error("NoEmoji() = false, want true (implied by AccessibilityMode)")
	}
}

func TestEmojiOutput(t *testing.T) {
	tests := []struct {
		name         string
		accessMode   bool
		noEmoji      bool
		emojiName    string
		expectedEmoji string
	}{
		{"Normal mode", false, false, "rocket", "🚀"},
		{"No emoji mode", false, true, "rocket", ""},
		{"Accessibility mode", true, false, "rocket", ""},
		{"Unknown emoji", false, false, "unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Language:          "en",
				AccessibilityMode: tt.accessMode,
				NoEmoji:           tt.noEmoji,
			}

			localizer, err := NewLocalizer(cfg)
			if err != nil {
				t.Fatalf("NewLocalizer() error = %v", err)
			}

			result := localizer.Emoji(tt.emojiName)
			if result != tt.expectedEmoji {
				t.Errorf("Emoji(%q) = %q, want %q", tt.emojiName, result, tt.expectedEmoji)
			}
		})
	}
}

func TestSymbolOutput(t *testing.T) {
	tests := []struct {
		name           string
		accessMode     bool
		noEmoji        bool
		symbolName     string
		expectedSymbol string
	}{
		{"Normal mode success", false, false, "success", "✅"},
		{"No emoji mode success", false, true, "success", "[✓]"},
		{"Accessibility mode success", true, false, "success", "[✓]"},
		{"Normal mode error", false, false, "error", "❌"},
		{"Accessibility mode error", true, false, "error", "[✗]"},
		{"Normal mode warning", false, false, "warning", "⚠️"},
		{"Accessibility mode warning", true, false, "warning", "[!]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Language:          "en",
				AccessibilityMode: tt.accessMode,
				NoEmoji:           tt.noEmoji,
			}

			localizer, err := NewLocalizer(cfg)
			if err != nil {
				t.Fatalf("NewLocalizer() error = %v", err)
			}

			result := localizer.Symbol(tt.symbolName)
			if result != tt.expectedSymbol {
				t.Errorf("Symbol(%q) = %q, want %q", tt.symbolName, result, tt.expectedSymbol)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		accMode  bool
		status   string
		message  string
		expected string
	}{
		{"Normal mode", false, "success", "Done", "✅ Done"},
		{"Accessibility mode", true, "success", "Done", "[✓] Done"},
		{"Error status", false, "error", "Failed", "❌ Failed"},
		{"Warning status", true, "warning", "Be careful", "[!] Be careful"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Language:          "en",
				AccessibilityMode: tt.accMode,
			}

			localizer, err := NewLocalizer(cfg)
			if err != nil {
				t.Fatalf("NewLocalizer() error = %v", err)
			}

			result := localizer.FormatStatus(tt.status, tt.message)
			if result != tt.expected {
				t.Errorf("FormatStatus(%q, %q) = %q, want %q",
					tt.status, tt.message, result, tt.expected)
			}
		})
	}
}

func TestGlobalInit(t *testing.T) {
	// Save original Global
	origGlobal := Global
	defer func() {
		Global = origGlobal
	}()

	err := Init(Config{
		Language: "en",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if Global == nil {
		t.Fatal("Global localizer is nil after Init()")
	}

	if Global.Language() != "en" {
		t.Errorf("Global.Language() = %q, want %q", Global.Language(), "en")
	}
}

func TestGlobalConvenienceFunctions(t *testing.T) {
	// Save original Global
	origGlobal := Global
	defer func() {
		Global = origGlobal
	}()

	// Initialize with English
	err := Init(Config{Language: "en"})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test Emoji
	emoji := Emoji("rocket")
	if emoji != "🚀" {
		t.Errorf("Emoji(\"rocket\") = %q, want %q", emoji, "🚀")
	}

	// Test Symbol
	symbol := Symbol("success")
	if symbol != "✅" {
		t.Errorf("Symbol(\"success\") = %q, want %q", symbol, "✅")
	}

	// Test FormatStatus
	status := FormatStatus("success", "Test message")
	if status != "✅ Test message" {
		t.Errorf("FormatStatus() = %q, want %q", status, "✅ Test message")
	}
}

func TestGlobalFunctionsWithoutInit(t *testing.T) {
	// Save original Global
	origGlobal := Global
	Global = nil
	defer func() {
		Global = origGlobal
	}()

	// Test that functions don't panic when Global is nil
	emoji := Emoji("rocket")
	if emoji != "" {
		t.Errorf("Emoji() without init = %q, want empty string", emoji)
	}

	symbol := Symbol("success")
	if symbol != "?" {
		t.Errorf("Symbol() without init = %q, want %q", symbol, "?")
	}

	// T should return key
	msg := T("test.key")
	if msg != "test.key" {
		t.Errorf("T() without init = %q, want %q", msg, "test.key")
	}
}

func TestSupportedLanguages(t *testing.T) {
	langs := SupportedLanguages()
	expected := []string{"en", "es", "fr", "de", "ja", "pt"}

	if len(langs) != len(expected) {
		t.Fatalf("SupportedLanguages() returned %d languages, want %d", len(langs), len(expected))
	}

	for i, lang := range expected {
		if langs[i] != lang {
			t.Errorf("SupportedLanguages()[%d] = %q, want %q", i, langs[i], lang)
		}
	}
}
