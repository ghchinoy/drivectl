---
title: Introduction to drivectl
description: Interact with Google Drive, Docs, and Sheets from your terminal or AI coding agent.
---

## What is drivectl?

`drivectl` is a command-line interface (CLI) and Model Context Protocol (MCP) server written in Go for interacting with **Google Drive**, **Google Docs**, and **Google Sheets**, as well as arbitrary Google Workspace REST endpoints via Google API Discovery Documents.

It is designed for both human developers working in an interactive shell and AI coding agents that require deterministic JSON output (`-O json`), proactive error hints, and structured tool schemas.

## Choose a starting point

- [Install](/install): download a prebuilt binary, install via `go install`, and configure your OAuth client secret.
- [Getting started](/getting-started): authenticate once and run your first Drive, Docs, and Sheets commands.
- [Architecture & philosophy](/architecture): learn when `drivectl` uses dedicated native subcommands versus dynamic Discovery calls.
- [Commands & Modes](/commands): explore the command reference, JSON recipes, and MCP server mode.

## Core capabilities

- **Ergonomic native subcommands:** Purpose-built commands for file uploads (`upload`), Markdown-to-Google-Docs AST conversion (`docs create`, `get --format md`), and Google Sheets A1 cell ranges (`sheets get-range`, `sheets update-range`).
- **Dynamic API Discovery (`call`):** Invoke any Google Workspace REST endpoint (`drive.v3.*`, `docs.v1.*`, `sheets.v4.*`) by passing a method identifier and a JSON `--payload`. `drivectl` inspects the live Discovery schema to route parameters to path variables, query strings, or the request body automatically.
- **Agent-native output & MCP server:** Pass `-O json` to any command for machine-readable output, or launch `drivectl --mcp` (stdio) / `drivectl --mcp-http :8080` to expose Workspace tools directly to MCP clients.

## Learn more

Read the [Design Philosophy](../../docs/DESIGN_PHILOSOPHY.md), browse the [bundled Agent Skill](../../skills/google-drive-cli/SKILL.md), or inspect the [release workflow](../../docs/RELEASING.md).
