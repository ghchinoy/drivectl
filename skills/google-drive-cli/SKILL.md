---
name: google-drive-cli
description: Interact with Google Drive, Docs, Sheets, and Slides using the drivectl CLI. Use this skill when asked to list/upload Drive files, read/update Sheets, create/export Google Docs (Markdown AST), or inspect/update Google Slides presentations.
metadata:
  discovery_doc_drive_v3: "https://www.googleapis.com/discovery/v1/apis/drive/v3/rest"
---
# google-drive-cli

This skill provides guidance on using the `drivectl` command-line tool to interact with Google Workspace (Drive, Docs, Sheets, Slides).

## Prerequisites & Authentication

- Ensure `drivectl` is available in your PATH or in the current working directory (e.g., `./drivectl`).
  *Install via Homebrew (`brew install ghchinoy/tap/drivectl`), Go (`go install github.com/ghchinoy/drivectl@latest`), or the bundled installer script `scripts/install.sh`.*
- **Crucial Rule:** Always append `-O json` to `drivectl` commands when you need to parse the output programmatically or return structured information.

### Authentication & Profiles

The user must authenticate once before using `drivectl`. Tokens are encrypted at rest (AES-256-GCM) in `~/.config/drivectl/` (or `~/.config/drivectl/profiles/<name>/` for named profiles) and refreshed automatically.

```bash
# Check current auth status, scopes, and encryption state (non-interactive)
drivectl auth status -O json

# Inspect minimal required scopes for a command before running it
drivectl auth scopes upload -O json

# Standard interactive login (PKCE S256 on ephemeral 127.0.0.1 loopback; supports --readonly, --services, --prompt-consent)
drivectl auth login --secret-file /path/to/your/client_secret.json

# Dedicated agentic OAuth profile (for clients marked "Use this client for an AI-powered agent")
drivectl auth login --agentic --secret-file /path/to/agent_client_secret.json

# Refresh, revoke/logout, or reset a profile
drivectl auth refresh
drivectl auth logout --profile agent
drivectl auth reset --profile agent

# Headless / CI execution with a pre-obtained OAuth access token (env var or --token-file)
DRIVECTL_TOKEN="$ACCESS_TOKEN" drivectl list -O json
```

*(Never run interactive `auth login` unattended inside a headless agent without user involvement; if operating on a remote/SSH host without a local browser, append `--no-browser-auth` so the user can open the authorization URL and paste back the code).*

## Available Modules

The `drivectl` tool provides subcommands organized by Workspace service. Pick the matching reference depending on the user's request:

- **Google Drive (Files/Folders/Uploads)**: See [references/drive.md](references/drive.md) for listing, querying, describing, downloading, uploading (with `--convert` and `--file-id` in-place updates), revisions, and comments.
- **Google Docs**: See [references/docs.md](references/docs.md) for creating docs from Markdown, exporting Docs to Markdown/PDF/text, and exploring tab hierarchies.
- **Google Sheets**: See [references/sheets.md](references/sheets.md) for reading A1 ranges, updating cells, and exporting sheets to RFC 4180 CSV.
- **Google Slides**: See [references/slides.md](references/slides.md) for creating decks, rendering presentations as Markdown, inspecting slide/element `objectId`s, replacing text, managing speaker notes, and converting `.pptx` files in place.

## Dynamic Discovery (Fallback)

If the built-in commands (`list`, `upload`, `docs`, `sheets`, `slides`) do not cover the requirement, use `drivectl call` to dynamically invoke *any* Google Workspace REST endpoint using Google Discovery (`--payload` or `--payload-file` / stdin).

See [references/discovery.md](references/discovery.md) for detailed instructions on formulating and executing dynamic API calls.
