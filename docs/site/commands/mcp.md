---
title: MCP server & Agent Skill
description: Expose Google Workspace tools to AI coding assistants over Model Context Protocol or the bundled Agent Skill.
---

## Start the MCP server

`drivectl` embeds a Model Context Protocol (MCP) server built on `github.com/modelcontextprotocol/go-sdk/mcp`.

### Stdio transport (default for local agents)

```sh
drivectl --mcp
```

### HTTP transport (Streamable HTTP + SSE)

```sh
drivectl --mcp-http :8080
```

In HTTP mode, `drivectl` exposes:
- `/mcp` — Streamable HTTP MCP endpoint
- `/sse` and `/messages` — Server-Sent Events compatibility endpoints

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
   - `slides.v1.presentations.get`

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
