---
title: Architecture & philosophy
description: How drivectl balances ergonomic native commands, dynamic API Discovery, and MCP tool generation.
---

## Generic Discovery vs. native subcommands

A central design principle in `drivectl` (documented in [Design Philosophy](../../docs/DESIGN_PHILOSOPHY.md)) is balancing the dynamic `call` command with dedicated native subcommands (`list`, `get`, `upload`, `docs`, `sheets`).

### When `call` is used

The [`drivectl call`](/drivectl/commands/discovery) command is a lightweight bridge to the Google API Discovery Service. You provide a dot-separated method identifier (`<service>.<version>.<resource>.<method>`) and a JSON `--payload`:

- **Uncomplicated REST/JSON:** Ideal for any Google Workspace endpoint whose request and response are pure JSON (`drive.v3.permissions.create`, `docs.v1.documents.create`, `drive.v3.files.update`).
- **Automatic parameter routing:** `internal/discovery` inspects the Discovery Document at runtime and routes payload fields to URL path templates, query parameters, or the HTTP request body.

:::caution[Permission Grants via Discovery vs. `share-with`]
Because `drivectl call` is a raw passthrough to Google Discovery, calling `drive.v3.permissions.create` without `"sendNotificationEmail": false` and `"supportsAllDrives": true` triggers Google Drive's default email notifications and fails on Shared Drives. For dedicated ACL management, use [`share-with`](https://github.com/ghchinoy/share-with) ([documentation](https://ghchinoy.github.io/share-with/)), which enforces `sendNotificationEmail=false` at the HTTP transport layer, parses `?resourcekey=...` URLs, and prevents accidental collaborator role downgrades.
:::

### When native subcommands are used

Whenever an operation involves binary streaming, multi-step protocols, or AST transformation, `drivectl` provides a dedicated subcommand under `cmd/` and `internal/drive/`:

- **Media streaming & uploads:** [`drivectl upload`](/drivectl/commands/drive) handles multipart media streams, parent folder placement, and custom MIME types cleanly.
- **Document AST conversion:** [`drivectl docs create`](/drivectl/commands/docs) parses local Markdown with Goldmark into Google Docs `BatchUpdate` requests, while `drivectl get --format md` walks the Google Docs structural AST (`Paragraph`, `Table`, `List`, `TextRun`) to reconstruct clean Markdown.
- **Spreadsheet range ergonomics:** [`drivectl sheets`](/drivectl/commands/sheets) wraps A1 range notation and CSV export without requiring verbose nested JSON payloads.

## Agent-aware CLI design

`drivectl` follows Agent-to-Agent (A2A) CLI conventions:

1. **Semantic command groups:** Subcommands are organized into `workspace` (Drive, Docs, Sheets), `api` (`call`, `run`), and `config` (`auth`).
2. **Deterministic JSON mode:** Passing `-O json` (`--output-format json`) suppresses terminal styling and emits structured JSON on stdout.
3. **Actionable error hints:** Authentication and API errors include remediation hints (such as re-running `drivectl auth login` or enabling the target API in Google Cloud Console).

## MCP schema normalization

When running in [MCP server mode](/drivectl/commands/mcp) (`--mcp` or `--mcp-http`), `drivectl` converts Google Discovery methods into JSON Schema tool definitions validated by `github.com/modelcontextprotocol/go-sdk`:

- **Typed parameter defaults:** Discovery parameter defaults (which Google publishes as strings such as `"false"` or `"100"`) are coerced into raw JSON booleans and numbers so `ValidateDefaults: true` succeeds.
- **`"any"` types & recursive `$ref`s:** Discovery `"type": "any"` is normalized to an unconstrained schema node, and recursive schema references (such as `docs.v1` `Tab.childTabs`) are tracked with a per-branch visited set to prevent cycles while preserving sibling `$ref` expansions.
