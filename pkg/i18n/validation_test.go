package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/BurntSushi/toml"
)

// supportedLanguages lists all languages that must have translations
var supportedLanguages = []string{"en", "es", "fr", "de", "ja", "pt"}

// translationEntry represents a single translation with metadata
type translationEntry struct {
	Description string `toml:"description"`
	Other       string `toml:"other"`
	One         string `toml:"one"`
	Zero        string `toml:"zero"`
	Many        string `toml:"many"`
}

// TestAllLanguagesHaveSameKeys validates that all language files have identical keys
func TestAllLanguagesHaveSameKeys(t *testing.T) {
	// Load English as the source of truth
	enKeys, err := loadTranslationKeys("en")
	if err != nil {
		t.Fatalf("Failed to load English translations: %v", err)
	}

	if len(enKeys) == 0 {
		t.Fatal("English translation file has no keys")
	}

	t.Logf("English has %d translation keys", len(enKeys))

	// Compare each language against English
	for _, lang := range supportedLanguages {
		if lang == "en" {
			continue
		}

		t.Run(lang, func(t *testing.T) {
			langKeys, err := loadTranslationKeys(lang)
			if err != nil {
				t.Fatalf("Failed to load %s translations: %v", lang, err)
			}

			// Check for missing keys in target language
			missing := []string{}
			for key := range enKeys {
				if _, exists := langKeys[key]; !exists {
					missing = append(missing, key)
				}
			}

			// Check for extra keys in target language
			extra := []string{}
			for key := range langKeys {
				if _, exists := enKeys[key]; !exists {
					extra = append(extra, key)
				}
			}

			if len(missing) > 0 {
				t.Errorf("%s is missing %d keys:\n  %v", lang, len(missing), missing[:min(10, len(missing))])
			}

			if len(extra) > 0 {
				t.Errorf("%s has %d extra keys:\n  %v", lang, len(extra), extra[:min(10, len(extra))])
			}

			if len(missing) == 0 && len(extra) == 0 {
				t.Logf("%s: ✓ All %d keys match English", lang, len(langKeys))
			}
		})
	}
}

// TestTemplateVariablesConsistent validates that template variables match across languages
func TestTemplateVariablesConsistent(t *testing.T) {
	// Template variable pattern: {{.Variable}}
	varPattern := regexp.MustCompile(`\{\{\.(\w+)\}\}`)

	enTranslations, err := loadTranslations("en")
	if err != nil {
		t.Fatalf("Failed to load English translations: %v", err)
	}

	for _, lang := range supportedLanguages {
		if lang == "en" {
			continue
		}

		t.Run(lang, func(t *testing.T) {
			langTranslations, err := loadTranslations(lang)
			if err != nil {
				t.Fatalf("Failed to load %s translations: %v", lang, err)
			}

			inconsistencies := 0

			for key, enEntry := range enTranslations {
				langEntry, exists := langTranslations[key]
				if !exists {
					continue // Already checked in TestAllLanguagesHaveSameKeys
				}

				// Extract variables from English
				enVars := extractVariables(varPattern, enEntry.Other)

				// Extract variables from target language
				langVars := extractVariables(varPattern, langEntry.Other)

				// Compare variable sets
				if !setsEqual(enVars, langVars) {
					inconsistencies++
					if inconsistencies <= 5 { // Only show first 5
						t.Errorf("Key '%s': English vars %v != %s vars %v",
							key, enVars, lang, langVars)
					}
				}

				// Check plural forms if they exist
				if enEntry.One != "" {
					enVarsOne := extractVariables(varPattern, enEntry.One)
					langVarsOne := extractVariables(varPattern, langEntry.One)
					if !setsEqual(enVarsOne, langVarsOne) {
						inconsistencies++
						if inconsistencies <= 5 {
							t.Errorf("Key '%s' (one): English vars %v != %s vars %v",
								key, enVarsOne, lang, langVarsOne)
						}
					}
				}
			}

			if inconsistencies == 0 {
				t.Logf("%s: ✓ All template variables consistent", lang)
			} else if inconsistencies > 5 {
				t.Errorf("... and %d more inconsistencies", inconsistencies-5)
			}
		})
	}
}

// TestTOMLSyntaxValid validates that all TOML files parse without errors
func TestTOMLSyntaxValid(t *testing.T) {
	for _, lang := range supportedLanguages {
		t.Run(lang, func(t *testing.T) {
			_, err := loadTranslations(lang)
			if err != nil {
				t.Errorf("Failed to parse %s TOML file: %v", lang, err)
			} else {
				t.Logf("%s: ✓ TOML syntax valid", lang)
			}
		})
	}
}

// TestNoMissingTranslations validates that no keys have missing translation strings
func TestNoMissingTranslations(t *testing.T) {
	for _, lang := range supportedLanguages {
		t.Run(lang, func(t *testing.T) {
			translations, err := loadTranslations(lang)
			if err != nil {
				t.Fatalf("Failed to load %s translations: %v", lang, err)
			}

			missing := 0
			for key, entry := range translations {
				if entry.Other == "" {
					missing++
					if missing <= 5 {
						t.Errorf("Key '%s' has empty 'other' translation", key)
					}
				}
			}

			if missing > 5 {
				t.Errorf("... and %d more missing translations", missing-5)
			}

			if missing == 0 {
				t.Logf("%s: ✓ No missing translations", lang)
			}
		})
	}
}

// TestTranslationKeyFormat validates that keys follow naming convention
func TestTranslationKeyFormat(t *testing.T) {
	// Keys should follow pattern: <cli>.<category>.<subcategory>.<identifier>
	// Examples: spawn.launch.short, truffle.search.error.invalid_pattern
	// Allows digits in key names (e.g., step2, g6_xlarge)
	keyPattern := regexp.MustCompile(`^[a-z]+\.[a-z0-9_]+(\.[a-z0-9_]+)*$`)

	enKeys, err := loadTranslationKeys("en")
	if err != nil {
		t.Fatalf("Failed to load English translations: %v", err)
	}

	invalid := 0
	for key := range enKeys {
		if !keyPattern.MatchString(key) {
			invalid++
			if invalid <= 10 {
				t.Errorf("Key '%s' doesn't match naming convention", key)
			}
		}
	}

	if invalid > 10 {
		t.Errorf("... and %d more invalid keys", invalid-10)
	}

	if invalid == 0 {
		t.Logf("✓ All %d keys follow naming convention", len(enKeys))
	}
}

// TestTranslationCoverage reports translation statistics
func TestTranslationCoverage(t *testing.T) {
	enTranslations, err := loadTranslations("en")
	if err != nil {
		t.Fatalf("Failed to load English translations: %v", err)
	}

	totalKeys := len(enTranslations)

	t.Logf("\n=== Translation Coverage Report ===")
	t.Logf("Total keys in English: %d", totalKeys)
	t.Logf("")

	for _, lang := range supportedLanguages {
		langTranslations, err := loadTranslations(lang)
		if err != nil {
			t.Logf("%s: ERROR loading file", lang)
			continue
		}

		coverage := (float64(len(langTranslations)) / float64(totalKeys)) * 100
		t.Logf("%s: %d keys (%.1f%% coverage)", lang, len(langTranslations), coverage)
	}
}

// Helper functions

// loadTranslations loads and parses a translation file using raw TOML parsing
func loadTranslations(lang string) (map[string]translationEntry, error) {
	filename := filepath.Join(".", fmt.Sprintf("active.%s.toml", lang))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse TOML into generic map to handle nested sections
	var raw map[string]interface{}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Flatten nested structure
	translations := make(map[string]translationEntry)
	flattenMap("", raw, translations)

	return translations, nil
}

// flattenMap recursively flattens a nested map structure from TOML
func flattenMap(prefix string, m map[string]interface{}, result map[string]translationEntry) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Check if this is a translation entry (has "other" field)
			if _, hasOther := v["other"]; hasOther {
				entry := translationEntry{}
				if desc, ok := v["description"].(string); ok {
					entry.Description = desc
				}
				if otherStr, ok := v["other"].(string); ok {
					entry.Other = otherStr
				}
				if one, ok := v["one"].(string); ok {
					entry.One = one
				}
				if zero, ok := v["zero"].(string); ok {
					entry.Zero = zero
				}
				if many, ok := v["many"].(string); ok {
					entry.Many = many
				}
				result[fullKey] = entry
			} else {
				// Nested section, recurse
				flattenMap(fullKey, v, result)
			}
		}
	}
}

// loadTranslationKeys loads only the keys from a translation file
func loadTranslationKeys(lang string) (map[string]bool, error) {
	translations, err := loadTranslations(lang)
	if err != nil {
		return nil, err
	}

	keys := make(map[string]bool, len(translations))
	for key := range translations {
		keys[key] = true
	}

	return keys, nil
}

// extractVariables extracts template variable names from a string
func extractVariables(pattern *regexp.Regexp, text string) []string {
	matches := pattern.FindAllStringSubmatch(text, -1)
	vars := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			vars = append(vars, match[1])
		}
	}
	return vars
}

// setsEqual checks if two string slices contain the same elements (order doesn't matter)
func setsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	setA := make(map[string]bool, len(a))
	for _, v := range a {
		setA[v] = true
	}

	for _, v := range b {
		if !setA[v] {
			return false
		}
	}

	return true
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
