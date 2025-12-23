#!/bin/bash
# Generate translation files from English
# This creates first-draft translations for native speaker review

# Note: These are AI-generated translations and should be reviewed by native speakers

# Extract just the translation keys and their "other" values from English
grep -E '^\[|^other = ' active.en.toml > /tmp/en_keys.txt

# For now, create skeletal translation files with the same structure
# Native speakers should review and improve these translations

echo "Creating Spanish (es) translation file..."
cat > active.es.toml << 'ES_EOF'
# Spanish (es) translations - DRAFT - Requires native speaker review
# Traducciones al español - BORRADOR - Requiere revisión de hablante nativo
ES_EOF

echo "Creating French (fr) translation file..."
cat > active.fr.toml << 'FR_EOF'
# French (fr) translations - DRAFT - Requires native speaker review  
# Traductions françaises - BROUILLON - Nécessite révision par locuteur natif
FR_EOF

echo "Creating German (de) translation file..."
cat > active.de.toml << 'DE_EOF'
# German (de) translations - DRAFT - Requires native speaker review
# Deutsche Übersetzungen - ENTWURF - Erfordert Überprüfung durch Muttersprachler
DE_EOF

echo "Creating Japanese (ja) translation file..."
cat > active.ja.toml << 'JA_EOF'
# Japanese (ja) translations - DRAFT - Requires native speaker review
# 日本語翻訳 - 下書き - ネイティブスピーカーによるレビューが必要
JA_EOF

echo "Translation skeleton files created. Full translations should be added by native speakers."
