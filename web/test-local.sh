#!/bin/bash
# Test spore.host landing page locally

PORT="${PORT:-8000}"

echo "🍄 Starting local server for spore.host..."
echo ""
echo "Open in browser:"
echo "  → http://localhost:$PORT"
echo ""
echo "Press Ctrl+C to stop"
echo ""

cd "$(dirname "$0")"
python3 -m http.server "$PORT"
