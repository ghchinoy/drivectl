---
title: Introduction to drivectl
description: Interact with Google Drive, Docs, Sheets, and Slides from your terminal or AI coding agent.
---

## What is drivectl?

:::note[Official Google Workspace CLI (`gwscli`) & Remote MCP servers]
Google now provides an official [Google Workspace CLI (`gwscli`)](https://developers.google.com/workspace/guides/workspace-cli) (prebuilt Go CLI over `*mcp.googleapis.com` for 9 services including Tasks) and hosted [Google Workspace Remote MCP servers](https://developers.google.com/workspace/guides/configure-mcp-servers#support-products) (10 endpoints adding Classroom and Universal Search), as well as the open-source Rust [`gws`](https://github.com/googleworkspace/cli) CLI. Start with those for general Workspace automation or multi-product workflows (Gmail, Calendar, Chat, Tasks, Universal Search).

`drivectl` remains a strong fit when you need an **embeddable Go library** (`pkg/drive`, `pkg/discovery`, `mcp.RegisterTools`), **stable versioned CLI flags for static scripts**, **Markdown $\leftrightarrow$ Google Docs** AST conversion, **Google Slides CLI ergonomics** (including in-place `.pptx` conversion), **zero usage telemetry**, or a single-binary local MCP server without Developer Preview enrollment.
:::

`drivectl` is a command-line interface (CLI) and Model Context Protocol (MCP) server written in Go for interacting with **Google Drive**, **Google Docs**, **Google Sheets**, and **Google Slides**, as well as arbitrary Google Workspace REST endpoints via Google API Discovery Documents.

It is designed for both human developers working in an interactive shell and AI coding agents that require deterministic JSON output (`-O json`), `--dry-run` write previews, proactive error hints, and annotated MCP tool schemas.

## Choose a starting point

- [Install](/drivectl/install): install via Homebrew (`brew install ghchinoy/tap/drivectl`), prebuilt binary, or `go install`, and configure your OAuth client secret (including 1-click Google Personal Mode).
- [Getting started](/drivectl/getting-started): authenticate once (PKCE + AES-256-GCM cache, least-privilege `--readonly`/`--services`, or `DRIVECTL_TOKEN`) and run your first Drive, Docs, Sheets, and Slides commands.
- [Architecture & philosophy](/drivectl/architecture): learn when `drivectl` uses dedicated native subcommands versus dynamic Discovery calls.
- [Commands & Modes](/drivectl/commands): explore the command reference, `--dry-run`, JSON recipes, and MCP server mode.

## Core capabilities

- **Ergonomic native subcommands:** Purpose-built commands for file uploads & in-place updates (`upload --file-id --convert`), Markdown-to-Google-Docs AST conversion (`docs create`, `get --format md`), Google Sheets A1 cell ranges & RFC 4180 CSV (`sheets get`, `sheets get-range`, `sheets update-range`), and Google Slides (`slides get`, `create`, `inspect`, `replace`, `notes`).
- **Dynamic API Discovery (`call`) & `--dry-run`:** Invoke any Google Workspace REST endpoint (`drive.v3.*`, `docs.v1.*`, `sheets.v4.*`, `slides.v1.*`) using `--payload` or `--payload-file`, and pass `--dry-run` to preview resolved HTTP requests without executing writes.
- **Agent-native output & annotated MCP server:** Pass `-O json` to any command for machine-readable output, or launch `drivectl --mcp` (stdio) / `drivectl --mcp-http :8080` (loopback-hardened HTTP) to expose 22 annotated Workspace tools (8 native AST/Slides/CSV tools + 14 Discovery tools) directly to MCP clients.

:::tip[Choosing between `drivectl` and `share-with`]
Use **`drivectl`** for Google Drive search, binary uploads, Markdown $\leftrightarrow$ Google Docs AST conversion, Sheets A1 cell ranges & CSV export, Google Slides authoring, dynamic Discovery calls, and MCP server mode.

For dedicated Google Drive sharing and ACL management, use the sister CLI [**`share-with`**](https://github.com/ghchinoy/share-with) ([documentation](https://ghchinoy.github.io/share-with/)), which enforces silent-only permission grants (`sendNotificationEmail=false` at the HTTP transport layer), parses `?resourcekey=...` links, and guards against accidental collaborator role downgrades.
:::

## Learn more

Read the [Design Philosophy](../../docs/DESIGN_PHILOSOPHY.md), browse the [bundled Agent Skill](../../skills/google-drive-cli/SKILL.md), or inspect the [release workflow](../../docs/RELEASING.md).
