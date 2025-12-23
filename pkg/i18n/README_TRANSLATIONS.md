# Translation Files - First Draft

## Overview

I've created first-draft translations for spawn and truffle CLIs in multiple languages. These are AI-generated translations that should be reviewed by native speakers before production use.

## Files Created

1. **active.es.toml** (Spanish) - ✅ COMPLETE - 1594 lines, ~450 keys
2. **active.fr.toml** (French) - 🚧 IN PROGRESS - Core translations
3. **active.de.toml** (German) - TO CREATE
4. **active.ja.toml** (Japanese) - TO CREATE  
5. **active.pt.toml** (Portuguese) - TO CREATE

## Spanish Translation (Complete)

The Spanish translation is the most comprehensive with full coverage of:
- All spawn commands (launch, connect, list, dns, extend, state)
- Complete wizard (6 steps with all options)
- All progress indicators
- All truffle commands (search, capacity, spot, quotas)
- All output headers and error messages
- Global flags and common strings

**Total Coverage**: ~450 translation keys covering 100% of user-facing strings

## Testing Translations

### Test Spanish Translation

```bash
# Build with Spanish
cd spawn
./bin/spawn --lang es launch

# Or set environment variable
export SPAWN_LANG=es
./bin/spawn launch

# Test wizard in Spanish
./bin/spawn launch --interactive

# Test truffle in Spanish
cd ../truffle
./bin/truffle --lang es search m7i.large
```

### Verify Emoji Handling

```bash
# Test with emoji
./bin/spawn --lang es launch

# Test without emoji
./bin/spawn --lang es --no-emoji launch

# Test accessibility mode
./bin/spawn --lang es --accessibility launch
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

## Next Steps

1. **Complete French translation** - Continue from line 597
2. **Create German translation** - Use Spanish as template
3. **Create Japanese translation** - Requires cultural adaptation
4. **Create Portuguese translation** - Similar to Spanish

5. **Native speaker review** for each language
6. **Test with actual CLI usage**
7. **Iterate based on feedback**

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

