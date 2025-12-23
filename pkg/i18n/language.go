package i18n

import (
	"os"
	"strings"
)

// DetectLanguage detects language from multiple sources
// Priority: SPAWN_LANG/TRUFFLE_LANG > System locale > Default (en)
func DetectLanguage() string {
	// 1. Check app-specific env vars
	if lang := os.Getenv("SPAWN_LANG"); lang != "" {
		return NormalizeLanguage(lang)
	}
	if lang := os.Getenv("TRUFFLE_LANG"); lang != "" {
		return NormalizeLanguage(lang)
	}

	// 2. Check system locale
	if lang, ok := DetectFromSystem(); ok {
		return NormalizeLanguage(lang)
	}

	// 3. Default to English
	return "en"
}

// DetectFromSystem detects language from system environment variables
func DetectFromSystem() (string, bool) {
	// Try LANG first (most common)
	if lang := os.Getenv("LANG"); lang != "" {
		return lang, true
	}

	// Try LC_ALL
	if lang := os.Getenv("LC_ALL"); lang != "" {
		return lang, true
	}

	// Try LANGUAGE (colon-separated list, take first)
	if lang := os.Getenv("LANGUAGE"); lang != "" {
		parts := strings.Split(lang, ":")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0], true
		}
	}

	return "", false
}

// NormalizeLanguage converts language variants to supported base languages
// Examples:
//   - en_US, en-US, en_GB -> en
//   - es_ES, es-MX -> es
//   - fr_FR, fr-CA -> fr
//   - de_DE, de-AT -> de
//   - ja_JP -> ja
func NormalizeLanguage(lang string) string {
	// Convert to lowercase
	lang = strings.ToLower(lang)

	// Remove encoding (e.g., en_US.UTF-8 -> en_US)
	lang = strings.Split(lang, ".")[0]

	// Remove country code (e.g., en_US -> en)
	lang = strings.Split(lang, "_")[0]

	// Remove country code (e.g., en-US -> en)
	lang = strings.Split(lang, "-")[0]

	// Validate against supported languages
	if IsSupported(lang) {
		return lang
	}

	// Default to English if unsupported
	return "en"
}
