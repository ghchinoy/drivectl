---
title: Getting started
description: Authenticate with Google OAuth 2.0 and run your first Drive, Docs, and Sheets workflows.
---

## Authenticate once

If you have not created a Desktop OAuth 2.0 `client_secret.json` yet, follow [Configure a Google OAuth client secret](/drivectl/install#configure-a-google-oauth-client-secret) first (including setting up the OAuth consent screen and adding your account as a Test User if your app is External in Testing mode).

Pass your `client_secret.json` file to `drivectl auth login` on your first run, or set the `DRIVE_SECRETS` environment variable:

```sh
# Pass the secret file via flag
drivectl auth login --secret-file /path/to/client_secret.json

# Or via the DRIVE_SECRETS environment variable
DRIVE_SECRETS=/path/to/client_secret.json drivectl auth login
```

`drivectl` caches your client secret (`client_secret.json`) and OAuth token (`token.json`) in `~/.config/drivectl/` and refreshes access tokens automatically on subsequent runs. When resolving the secret path, `--secret-file` takes precedence over `DRIVE_SECRETS`, which takes precedence over `~/.config/drivectl/client_secret.json`.

:::note[Local OAuth callback on `http://localhost:8080`]
During default browser-based login, `drivectl` opens your browser and starts a temporary local HTTP listener on `http://localhost:8080` to receive the authorization callback. Make sure port `8080` is free while authenticating (for example, stop any local dev server or `drivectl --mcp-http :8080` first).
:::

If you are working on a remote or headless machine without a local browser, pass `--no-browser-auth` to print an authorization URL you can open on any device:

```sh
drivectl auth login --secret-file /path/to/client_secret.json --no-browser-auth
```

After you approve access in the browser, it will redirect to `http://localhost/?state=...&code=...` (a "cannot connect" page is expected because nothing is listening locally). Copy the `code` query parameter value from the browser address bar and paste it into the terminal prompt.

:::tip[Re-authenticating & `Error 403: access_denied`]
- Once `client_secret.json` is cached in `~/.config/drivectl/`, run `drivectl auth login` without `--secret-file` whenever you need a fresh consent flow (for example, to pick up newly added scopes or renew a 7-day Testing-mode refresh token).
- If Google shows `Error 403: access_denied` during consent, add your Google account under **Google Auth Platform > Audience > Test users** in the Google Cloud Console.
:::

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

Explore the full [command reference](/drivectl/commands) or connect `drivectl` to an AI assistant in [MCP server & Agent Skill](/drivectl/commands/mcp).
