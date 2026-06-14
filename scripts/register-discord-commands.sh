#!/usr/bin/env bash
# Register spore-bot's global Discord slash commands (one-time, per application).
#
# Discord application commands are registered once on the application; they then
# appear in every guild that installs the app. Re-run after changing the command
# shape. Globals can take up to ~1h to propagate (guild-scoped is instant — see
# below for testing).
#
# Usage:
#   DISCORD_APP_ID=...  DISCORD_BOT_TOKEN=...  ./register-discord-commands.sh
#   # fast iteration in ONE test guild (instant propagation):
#   DISCORD_APP_ID=...  DISCORD_BOT_TOKEN=...  DISCORD_TEST_GUILD_ID=...  ./register-discord-commands.sh
#
# Models `/spore <command> [name] [arg]` as a single command with options, so it
# maps onto the same {command,nickname,arg} the handler parses for Slack/Teams.
set -euo pipefail

: "${DISCORD_APP_ID:?set DISCORD_APP_ID (Application ID)}"
: "${DISCORD_BOT_TOKEN:?set DISCORD_BOT_TOKEN (Bot token)}"
CMD_NAME="${SPORE_COMMAND_NAME:-spore}"

if [[ -n "${DISCORD_TEST_GUILD_ID:-}" ]]; then
  URL="https://discord.com/api/v10/applications/${DISCORD_APP_ID}/guilds/${DISCORD_TEST_GUILD_ID}/commands"
  echo "Registering /${CMD_NAME} in test guild ${DISCORD_TEST_GUILD_ID} (instant)…"
else
  URL="https://discord.com/api/v10/applications/${DISCORD_APP_ID}/commands"
  echo "Registering GLOBAL /${CMD_NAME} (may take up to ~1h to appear everywhere)…"
fi

# Option type 3 = STRING. `command` is required; name/arg optional.
read -r -d '' PAYLOAD <<JSON || true
{
  "name": "${CMD_NAME}",
  "type": 1,
  "description": "Control your spore.host instances",
  "options": [
    { "name": "command", "description": "list, status, start, stop, hibernate, url, extend, connect, help", "type": 3, "required": true },
    { "name": "name", "description": "instance nickname (e.g. rstudio)", "type": 3, "required": false },
    { "name": "arg", "description": "extra argument (e.g. a duration for extend)", "type": 3, "required": false }
  ]
}
JSON

curl -sS -X POST "$URL" \
  -H "Authorization: Bot ${DISCORD_BOT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD" | { command -v jq >/dev/null 2>&1 && jq . || cat; }

echo
echo "Done. Set the application's Interactions Endpoint URL to the spore-bot Function URL + /discord."
