---
title: Getting started
description: Authenticate with Google OAuth 2.0 and run your first Drive, Docs, and Sheets workflows.
---

## Authenticate once

Pass your `client_secret.json` file to `drivectl auth login` on your first run:

```sh
drivectl auth login --secret-file /path/to/client_secret.json
```

`drivectl` caches your credentials and OAuth token in `~/.config/drivectl/` and refreshes access tokens automatically on subsequent runs.

If you are working on a remote or headless machine without a local browser, pass `--no-browser-auth` to print an authorization URL you can open on any device:

```sh
drivectl auth login --secret-file /path/to/client_secret.json --no-browser-auth
```

## List and search Drive files

List files in your Google Drive or filter them with Drive query syntax (`-q`):

```sh
# List the first 20 files in human-readable table format
drivectl list --limit 20

# Search for Google Docs matching a name and emit structured JSON
drivectl list -q "name contains 'Design' and mimeType='application/vnd.google-apps.document'" -O json
```

## Convert between Markdown and Google Docs

Create a formatted Google Doc directly from a local Markdown file, or export an existing Google Doc back to Markdown:

```sh
# Create a Google Doc from local Markdown
drivectl docs create "Project Proposal" ./docs/DESIGN_PHILOSOPHY.md -O json

# Export a Google Doc to local Markdown
drivectl get <document-id> --format md -o proposal.md
```

## Read and update Google Sheets

Export a sheet to CSV or read and write specific A1 cell ranges:

```sh
# Read cells A1:C5 as a JSON array of rows
drivectl sheets get-range <spreadsheet-id> --sheet "Sheet1" --range "A1:C5" -O json

# Update cell B2
drivectl sheets update-range <spreadsheet-id> "Approved" --sheet "Sheet1" --range "B2" -O json
```

## Next steps

Explore the full [command reference](/commands) or connect `drivectl` to an AI assistant in [MCP server & Agent Skill](/commands/mcp).
