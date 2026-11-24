---
title: MCP server & Agent Skill
description: Expose Google Workspace tools to AI coding assistants over Model Context Protocol or the bundled Agent Skill.
---

:::note[Official Google-hosted Workspace MCP servers]
Google now offers hosted remote MCP servers for [Drive, Docs, Sheets, Slides, Gmail, Calendar, Chat, People, Classroom, and Universal Search](https://developers.google.com/workspace/guides/configure-mcp-servers#support-products) (Developer Preview). Prefer those endpoints for managed, zero-ops agent connections; use `drivectl --mcp` or `mcp.RegisterTools` when you want a local single-binary MCP server without Developer Preview enrollment or need to embed Discovery-backed tools inside a Go MCP server.
:::

## Start the MCP server

`drivectl` embeds a Model Context Protocol (MCP) server built on `github.com/modelcontextprotocol/go-sdk/mcp`.

### Stdio transport (default for local agents)

```sh
drivectl --mcp
```

### HTTP transport (Streamable HTTP)

```sh
# Defaults hostless addresses (":8080") to 127.0.0.1:8080 with Origin validation
drivectl --mcp-http :8080

# Require Bearer token authentication on HTTP MCP requests
DRIVECTL_MCP_AUTH_TOKEN="your-secret-token" drivectl --mcp-http 127.0.0.1:8080
```

In HTTP mode, `drivectl` binds `127.0.0.1` by default, rejects non-loopback browser `Origin` headers (preventing DNS rebinding), and enforces `Authorization: Bearer <token>` whenever `DRIVECTL_MCP_AUTH_TOKEN` is set. When a dedicated `agent` profile (created via `drivectl auth login --agentic`) or `DRIVECTL_AGENT_SECRETS` is present, MCP mode selects it automatically.

## Registered MCP tools

When the server starts, it registers two tiers of tools:

1. **Native Workspace tools:**
   - `drivectl_list`: list files in Google Drive with optional query (`q`) and `pageSize`.
   - `drivectl_read`: read or export a Google Drive file or Google Doc (`raw`, `txt`, `md`).
   - `drivectl_call`: invoke any Google API method dynamically by `serviceMethod` and JSON `payload`.
2. **Dynamic Discovery tools:**
   - `drive.v3.files.list`, `drive.v3.files.get`, `drive.v3.files.create`, `drive.v3.files.update`
   - `docs.v1.documents.create`, `docs.v1.documents.get`, `docs.v1.documents.batchUpdate`
   - `sheets.v4.spreadsheets.create`, `sheets.v4.spreadsheets.get`, `sheets.v4.spreadsheets.values.get`, `sheets.v4.spreadsheets.values.update`
   - `slides.v1.presentations.create`, `slides.v1.presentations.get`, `slides.v1.presentations.batchUpdate`

## Embed in another Go MCP server

Use `mcp.RegisterTools` to mount `drivectl`'s resources and Discovery tools onto an existing `*mcp.Server` with a custom `ClientProvider`:

```go
err := drivectlmcp.RegisterTools(ctx, server, func(ctx context.Context) (*http.Client, error) {
    return drive.NewClientFromBearerToken(ctx, accessToken), nil
})
```

## Configure Gemini CLI

Copy [settings.json.sample](../../../settings.json.sample) to `.gemini/settings.json` in your workspace and point `command` to your `drivectl` binary:

```json
{
  "mcpServers": {
    "drivectl": {
      "command": "/usr/local/bin/drivectl",
      "args": ["--mcp"]
    }
  }
}
```

## Install the bundled Agent Skill

The repository includes the [google-drive-cli Agent Skill](../../../skills/google-drive-cli/SKILL.md) with progressive-disclosure references for [Drive](../../../skills/google-drive-cli/references/drive.md), [Docs](../../../skills/google-drive-cli/references/docs.md), [Sheets](../../../skills/google-drive-cli/references/sheets.md), and [Discovery](../../../skills/google-drive-cli/references/discovery.md).

```sh
# Install via Open Agent Skills CLI
npx skills add ghchinoy/drivectl --skill google-drive-cli
```
