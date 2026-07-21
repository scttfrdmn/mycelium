---
description: "The spore.host MCP server exposes truffle and spawn as tools for AI assistants that support the Model Context Protocol — including Claude Desktop and Cursor."
---

# MCP Server <span class="doc-badge automation">Automation</span> <span class="doc-badge stable">Stable</span>

**What it is.** The spore.host MCP server exposes truffle and spawn as tools for AI
assistants that support the [Model Context Protocol](https://modelcontextprotocol.io)
— including Claude Desktop and Cursor.

**When to use it.** When you'd rather describe what you need in plain language —
*"find the cheapest A100 in us-east-1, then give me the spawn command to launch it
with a 4-hour limit"* — and let the assistant do the search and manage running
instances, instead of typing every command yourself. It runs locally with your
AWS credentials, the same trust model as the CLIs.

::: warning The MCP server cannot launch instances — by design
The exposed tools are **read + manage-existing** only: search / spot prices /
quotas (truffle) and list / status / stop / terminate / extend (spawn). There is
**no `launch` tool**, deliberately: creating billable infrastructure from an AI
assistant is a boundary we don't cross automatically. The assistant helps you find
the right instance and *construct* the `spawn launch` command; you run it. See the
full tool list below.
:::

## Install

```sh
brew install spore-host/tap/spore-host-mcp
```

## Configure

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "spore-host": {
      "command": "/usr/local/bin/spore-host-mcp"
    }
  }
}
```

Restart Claude Desktop. A hammer icon in the input bar confirms the server is connected.

## Available tools

### Truffle tools

| Tool | Description |
|------|-------------|
| `truffle_find` | Natural language instance type search — `"nvidia h100 8gpu"`, `"cheap arm64 with 32gb"` |
| `truffle_spot_prices` | Current Spot prices for a specific type across regions and AZs |
| `truffle_quota_check` | Whether your account has sufficient quota to launch a type |

### Spawn tools

| Tool | Description |
|------|-------------|
| `spawn_list` | List instances, filter by state and region |
| `spawn_status` | Detailed status by instance name or ID |
| `spawn_stop` | Stop or hibernate a running instance |
| `spawn_terminate` | Permanently terminate an instance |
| `spawn_extend` | Update an instance's TTL |

## Example interactions

```
"What instances do I have running and how long until they terminate?"

"Find me the cheapest GPU instance for inference in us-east-1."

"Stop the rstudio instance — I forgot it was running."

"What's the current Spot price for p4d.24xlarge across us-east-1 and us-west-2?"

"Extend the bert-training TTL by 6 hours."
```

## Credentials

The MCP server uses whichever AWS credentials are active in your environment — the same ones the CLI uses. No additional setup is needed.

For a full setup walkthrough, see [AI Assistant (MCP)](/guides/mcp-setup).

## Source

The MCP server is open source and ships as a single static binary. → [spore-host-mcp on GitHub](https://github.com/spore-host/spore-host-mcp)
