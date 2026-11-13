---
title: Commands & modes
description: Explore drivectl's native Workspace subcommands, dynamic API Discovery caller, recipes, and MCP server mode.
---

## Workspace commands

Purpose-built subcommands for common Google Workspace file and document operations.

- [Google Drive](/drivectl/commands/drive) `list · describe · get · upload · revisions · comments`

  Search Drive files, inspect metadata, stream uploads and downloads, and read revision history or threaded comments.

- [Google Docs](/drivectl/commands/docs) `docs create · docs tabs · get --format md`

  Convert local Markdown into richly formatted Google Docs, export Docs back to Markdown/PDF/text, and inspect document tab hierarchies.

- [Google Sheets](/drivectl/commands/sheets) `sheets get · sheets get-range · sheets update-range`

  Export sheets to CSV, read cell values via A1 notation, and update cells from the command line.

## Dynamic API & automation

Call any Google Workspace REST method or chain commands into repeatable workflows.

- [Dynamic Discovery (`call`)](/drivectl/commands/discovery) `call · run`

  Invoke arbitrary Google API endpoints via live Discovery Documents and execute multi-step JSON recipes.

## AI agent integration

Expose `drivectl` directly to AI coding assistants via MCP or progressive-disclosure skills.

- [MCP server & Agent Skill](/drivectl/commands/mcp) `--mcp · --mcp-http · google-drive-cli`

  Run `drivectl` as a stdio or HTTP Model Context Protocol server and install the bundled `google-drive-cli` Agent Skill.
