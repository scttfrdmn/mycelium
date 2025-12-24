# Translation Files

## Overview

Complete translations for spawn and truffle CLIs in 6 languages. All translations have been validated for consistency, template variables, and 100% key coverage.

## Files Status

1. **active.en.toml** (English) - ✅ COMPLETE - 443 keys (source)
2. **active.es.toml** (Spanish) - ✅ COMPLETE - 443 keys (100%)
3. **active.fr.toml** (French) - ✅ COMPLETE - 443 keys (100%)
4. **active.de.toml** (German) - ✅ COMPLETE - 443 keys (100%)
5. **active.ja.toml** (Japanese) - ✅ COMPLETE - 443 keys (100%)
6. **active.pt.toml** (Portuguese) - ✅ COMPLETE - 443 keys (100%)

**All languages validated**: ✅ Keys match, ✅ Template variables consistent, ✅ TOML syntax valid

## Translation Coverage

All 6 languages have complete coverage of:
- All spawn commands (launch, connect, list, dns, extend, state, etc.)
- Complete interactive wizard (6 steps with all options)
- All progress indicators and status messages
- All truffle commands (search, capacity, spot, quotas)
- All output headers, tables, and error messages
- Global flags and common strings
- Agent monitoring and warnings

**Total Coverage**: 443 translation keys covering 100% of user-facing strings across all languages

## Testing Translations

### Automated Test Suite

Run the comprehensive test suite to validate all translations:

```bash
cd pkg/i18n

# Run all tests (validation + integration)
go test -v .

# Run only validation tests
go test -v -run "TestAllLanguagesHaveSameKeys|TestTemplateVariablesConsistent|TestTOMLSyntaxValid|TestNoMissingTranslations|TestTranslationKeyFormat|TestTranslationCoverage"

# Run only integration tests
go test -v -run "TestLanguageDetectionFromEnv|TestLanguageSwitchAtRuntime|TestEmojiAccessibilityMode|TestTranslationWithTemplateData"
```

**Test Coverage**:
- ✅ All languages have identical keys (443 each)
- ✅ Template variables consistent across translations
- ✅ TOML syntax valid in all files
- ✅ No missing translations
- ✅ Key naming convention followed
- ✅ Language detection from environment variables
- ✅ Runtime language switching
- ✅ Emoji and accessibility modes
- ✅ Template substitution with variables
- ✅ Pluralization support
- ✅ Error translation wrapping

### Manual CLI Testing

Test translations with actual CLI commands:

```bash
# Test with different languages
spawn --lang es launch --help    # Spanish
spawn --lang fr list --help      # French
spawn --lang de connect --help   # German
spawn --lang ja extend --help    # Japanese
spawn --lang pt state --help     # Portuguese

# Or set environment variable
export LANG=es_ES.UTF-8
spawn launch --interactive

# Test language auto-detection
LANG=fr_FR spawn launch --help
LANG=de_DE truffle search m7i.large
LANG=ja_JP spawn list
LANG=pt_BR truffle capacity --region us-east-1
```

### Test Emoji and Accessibility

```bash
# Test with emoji (default)
spawn --lang es launch

# Test without emoji
spawn --lang fr --no-emoji launch

# Test accessibility mode (screen reader friendly)
spawn --lang de --accessibility launch

# Accessibility mode implies no emoji
spawn --lang ja --accessibility list
```

### Test Template Variable Substitution

Translations use template variables like `{{.Variable}}`. Test that these work:

```bash
# Region prompts use {{.DefaultRegion}}
spawn --lang es launch --interactive

# Error messages use various template variables
spawn --lang fr connect invalid-instance-id

# Progress messages use {{.InstanceID}}, {{.PublicIP}}, etc.
spawn --lang de launch --instance-type m7i.large --region us-west-2
```

## Recommendations for Native Speaker Review

### Spanish Review Points

1. **Formal vs Informal "You"**
   - Current: Mixed usage (tú/usted)
   - Recommendation: Choose consistent voice

2. **Technical Term Choices**
   - "Instancia" vs "Ejemplar" for "Instance"
   - "Lanzar" vs "Iniciar" for "Launch"
   - "Terminación" vs "Finalización" for "Termination"

3. **Regional Variations**
   - Current translations lean towards neutral Spanish
   - May need regional adaptations for Latin America vs Spain

4. **AWS Term Consistency**
   - Verify AWS official Spanish terminology
   - Region names (Virginia del Norte vs North Virginia)

### General Review Points for All Languages

1. **Command names** - Keep in English or translate?
   - Current: Kept in English (launch, connect, list)
   - Alternative: Translate to native language

2. **Flag names** - Keep in English?
   - Current: Kept in English (--region, --instance-type)
   - This matches standard CLI conventions

3. **Error messages** - Technical detail level
   - Current: Direct translations
   - May need cultural adaptation

## Contributing Translations

### Native Speaker Review

While all translations are complete and validated for consistency, native speakers can help improve quality:

**How to contribute**:

1. **Review existing translations** in `active.*.toml`
2. **Check for:**
   - Natural phrasing in target language
   - Appropriate formality level
   - Accurate technical terminology
   - Regional variations (e.g., Latin America vs Spain Spanish)
   - Cultural appropriateness
3. **Submit improvements** via pull request
4. **Add context** in PR description about changes

### Translation Quality Checklist

When reviewing or updating translations:

- [ ] All keys present (compare to active.en.toml)
- [ ] Template variables match exactly ({{.Variable}})
- [ ] TOML syntax valid (run `go test -v .`)
- [ ] Formality consistent throughout
- [ ] Technical terms accurate
- [ ] Natural phrasing (not machine translation)
- [ ] Cultural appropriateness
- [ ] Regional variations considered

### Running Tests Before Submitting

Always run the test suite before submitting translation changes:

```bash
cd pkg/i18n

# Run all validation tests
go test -v .

# Check for specific issues
go test -v -run TestAllLanguagesHaveSameKeys        # Verify all keys present
go test -v -run TestTemplateVariablesConsistent    # Check {{.Variable}} usage
go test -v -run TestTOMLSyntaxValid                # Validate TOML format
```

### Adding New Translation Keys

When adding new features that need translations:

1. **Add to English first** (`active.en.toml`)
2. **Add to all other languages** (es, fr, de, ja, pt)
3. **Run validation tests** to ensure consistency
4. **Update this README** if adding new sections

Example:

```toml
# active.en.toml
[spawn.newfeature.title]
description = "Title for new feature"
other = "New Feature: {{.FeatureName}}"

# active.es.toml (and all other languages)
[spawn.newfeature.title]
description = "Title for new feature"
other = "Nueva Función: {{.FeatureName}}"
```

## File Structure

Each translation file follows this structure:

```toml
# Language header with metadata
[common.*]          # Shared strings
[spawn.root.*]      # Root command
[spawn.launch.*]    # Launch command
[spawn.wizard.*]    # Interactive wizard
[spawn.progress.*]  # Progress indicators
[spawn.connect.*]   # Connect command
[spawn.list.*]      # List command
[spawn.dns.*]       # DNS management
[spawn.extend.*]    # TTL extension
[spawn.state.*]     # State commands
[spawn.agent.*]     # Agent warnings
[truffle.root.*]    # Truffle root
[truffle.search.*]  # Search command
[truffle.capacity.*] # Capacity command
[truffle.spot.*]    # Spot pricing
[truffle.quotas.*]  # Service quotas
[truffle.output.*]  # Output formatting
[flags.*]           # Global flags
[error.*]           # Error messages
```

## Translation Quality Notes

### Strengths
- Comprehensive coverage of user-facing strings
- Preserved technical terminology  
- Maintained template variable syntax
- Consistent naming conventions

### Areas for Improvement (Native Speaker Review)
- Idiomatic expressions
- Cultural appropriateness
- Regional variations
- Technical term accuracy
- Tone consistency (formal vs informal)

##Command Examples in Multiple Languages

### Spanish
```bash
spawn --lang es launch --interactive
# Asistente de configuración spawn
# Paso 1 de 6: Elige el tipo de instancia
```

### French (when complete)
```bash
spawn --lang fr launch --interactive
# Assistant de configuration spawn
# Étape 1 sur 6 : Choisissez le type d'instance
```

### German (when complete)
```bash
spawn --lang de launch --interactive
# spawn-Konfigurationsassistent
# Schritt 1 von 6: Wählen Sie den Instanztyp
```

### Japanese (when complete)
```bash
spawn --lang ja launch --interactive
# spawn設定ウィザード
# ステップ1/6: インスタンスタイプを選択
```

### Portuguese (when complete)
```bash
spawn --lang pt launch --interactive
# Assistente de configuração spawn
# Passo 1 de 6: Escolha o tipo de instância
```

