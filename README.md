# drivectl

[![Go Version](https://img.shields.io/github/go-mod/go-version/ghchinoy/drivectl)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/ghchinoy/drivectl)](https://github.com/ghchinoy/drivectl/releases)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

A Go command-line interface (CLI) and Model Context Protocol (MCP) server for interacting with **Google Drive**, **Google Docs**, and **Google Sheets**, plus dynamic calls to any Google Workspace REST endpoint via Google API Discovery Documents. Built for both interactive terminal workflows and AI coding agents with deterministic JSON output (`-O json`), Markdown-to-Docs AST conversion, and multi-step JSON recipes.

> [!NOTE]
> For active production Google Workspace integrations, also see the [Gemini CLI Workspace Extension](https://github.com/gemini-cli-extensions/workspace) and the [Google Workspace CLI (`gws`)](https://github.com/googleworkspace/cli).

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Getting a Google OAuth Client Secret](#getting-a-google-oauth-client-secret)
- [Usage & Command Reference](#usage--command-reference)
  - [Authentication (`auth`)](#authentication-auth)
  - [Google Drive Files (`list`, `describe`, `get`, `upload`, `revisions`, `comments`)](#google-drive-files-list-describe-get-upload-revisions-comments)
  - [Google Docs (`docs`)](#google-docs-docs)
  - [Google Sheets (`sheets`)](#google-sheets-sheets)
  - [Dynamic Discovery Calls (`call`)](#dynamic-discovery-calls-call)
  - [Automating Workflows (`run`)](#automating-workflows-run)
- [MCP Server Mode](#mcp-server-mode)
- [Agent Skill & Gemini CLI Configuration](#agent-skill--gemini-cli-configuration)
- [Local Development](#local-development)
- [Releasing & Publishing](#releasing--publishing)
- [Contributing](#contributing)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Features

- **Robust Authentication:** OAuth 2.0 login with automatic local credential caching (`~/.config/drivectl/`), silent token refresh, and headless `--no-browser-auth` support.
- **Agent-Friendly UX:** Semantic command grouping (`workspace`, `api`, `config`), color-coded terminal output, proactive error hints, and deterministic JSON output (`-O json`).
- **Dynamic API Discovery (`call`):** Invoke *any* Google Workspace REST endpoint (`drive.v3.*`, `docs.v1.*`, `sheets.v4.*`, etc.) dynamically from live Google API Discovery Documents.
- **Google Drive Operations:** List files with Drive query syntax (`-q`), inspect file metadata (`describe`), upload local files with custom MIME types (`upload`), download or export content (`get`), and inspect revision history (`revisions`) and threaded comments (`comments`).
- **Google Docs AST Conversion:** Convert local Markdown files into formatted Google Docs (`docs create`), export Google Docs back to Markdown, PDF, or plain text (`get --format md|pdf|txt`), and inspect document tab hierarchies (`docs tabs`).
- **Google Sheets Operations:** Export sheets to CSV (`sheets get`), read cell ranges via A1 notation (`sheets get-range`), and update cell values (`sheets update-range`).
- **Composable Recipes (`run`):** Execute sequential multi-step CLI workflows defined in JSON files.
- **Dual-Mode MCP Server (`--mcp` / `--mcp-http`):** Expose native and Discovery-driven Google Workspace tools (`drive_v3_files_list`, `docs_v1_documents_get`, `sheets_v4_spreadsheets_get`) over stdio or HTTP.

## Installation

### Prebuilt Binaries (macOS & Linux)

Download and verify the latest release binary into your current directory using the bundled installer script ([`skills/google-drive-cli/scripts/install.sh`](skills/google-drive-cli/scripts/install.sh)):

```bash
curl -sL https://raw.githubusercontent.com/ghchinoy/drivectl/main/skills/google-drive-cli/scripts/install.sh | bash
```

### Go Install

Requires **Go 1.26+**:

```bash
go install github.com/ghchinoy/drivectl@latest
```

### Build from Source

```bash
git clone https://github.com/ghchinoy/drivectl.git
cd drivectl
go build -o drivectl .
```

## Quick Start

Once installed and provided with a Desktop OAuth `client_secret.json` (see [Getting a Google OAuth Client Secret](#getting-a-google-oauth-client-secret)):

```bash
# 1. Authenticate once (caches credentials & token in ~/.config/drivectl/)
./drivectl auth login --secret-file /path/to/client_secret.json

# 2. List recent Google Docs as structured JSON
./drivectl list -q "mimeType='application/vnd.google-apps.document'" --limit 5 -O json

# 3. Export a Google Doc to local Markdown
./drivectl get <document-id> --format md -o notes.md
```

## Getting a Google OAuth Client Secret

1. Open the [Google Cloud Console](https://console.cloud.google.com/) and create or select a project.
2. Navigate to **APIs & Services > Library** and enable:
   - **Google Drive API** (`drive.googleapis.com`)
   - **Google Docs API** (`docs.googleapis.com`)
   - **Google Sheets API** (`sheets.googleapis.com`)
3. Navigate to **APIs & Services > Credentials**.
4. Click **Create Credentials > OAuth client ID** and select **Desktop app** as the application type.
5. Download the JSON file, save it as `client_secret.json`, and pass it once to `drivectl auth login --secret-file client_secret.json`.

## Usage & Command Reference

| Command | Group | Purpose |
|---|---|---|
| `drivectl auth login` | `config` | Authenticate with Google OAuth 2.0 and cache tokens locally |
| `drivectl list` | `workspace` | List and search files in Google Drive |
| `drivectl describe <file-id>` | `workspace` | Fetch detailed metadata for a Drive file |
| `drivectl get <file-id>` | `workspace` | Download a file or export a Google Doc (`raw`, `txt`, `md`, `pdf`) |
| `drivectl upload <local-path>` | `workspace` | Upload a local file to Google Drive |
| `drivectl revisions <file-id>` | `workspace` | List the revision history of a file |
| `drivectl comments <file-id>` | `workspace` | List threaded comments and replies on a file |
| `drivectl docs create <title> [md]` | `workspace` | Create a Google Doc (optionally converting a local Markdown file) |
| `drivectl docs tabs <doc-id>` | `workspace` | Inspect the structural hierarchy of tabs in a Google Doc |
| `drivectl sheets get <id>` | `workspace` | Export a Google Sheet tab to CSV |
| `drivectl sheets get-range <id>` | `workspace` | Read cell values from an A1 range |
| `drivectl sheets update-range <id> <val>` | `workspace` | Write a value to an A1 range |
| `drivectl call <api.ver.res.method>` | `api` | Dynamically invoke any Google REST endpoint via Discovery |
| `drivectl run <recipe.json>` | `api` | Execute a sequence of `drivectl` commands from a JSON recipe |
| `drivectl --mcp` / `--mcp-http :8080` | root flags | Run `drivectl` as an MCP server over stdio or HTTP |

### Authentication (`auth`)

`drivectl` caches your client secret and OAuth token in `~/.config/drivectl/` and refreshes tokens automatically.

```bash
# Browser-based OAuth login (default)
./drivectl auth login --secret-file /path/to/client_secret.json

# Headless / remote session login (prints manual authorization URL)
./drivectl auth login --secret-file /path/to/client_secret.json --no-browser-auth
```

### Google Drive Files (`list`, `describe`, `get`, `upload`, `revisions`, `comments`)

#### List and search files

```bash
# List the first 20 files
./drivectl list --limit 20

# Search using Google Drive query syntax and emit JSON
./drivectl list -q "name contains 'Proposal' and trashed = false" -O json
```

#### Describe file metadata and download content

```bash
# Inspect full file metadata
./drivectl describe <file-id> -O json

# Download raw file content to stdout
./drivectl get <file-id>

# Export a Google Doc as Markdown or PDF
./drivectl get <google-doc-id> --format md -o my-document.md
./drivectl get <google-doc-id> --format pdf -o my-document.pdf
```

#### Upload local files

```bash
# Upload a file to the root of My Drive
./drivectl upload ./report.pdf --name "Q3 Report.pdf"

# Upload into a specific folder with an explicit MIME type
./drivectl upload ./data.csv --parent <folder-id> --mime-type text/csv -O json
```

#### View file revisions and comments

```bash
# View revision history
./drivectl revisions <file-id> -O json

# View threaded comments
./drivectl comments <file-id> -O json
```

### Google Docs (`docs`)

#### Create a Google Doc from Markdown

```bash
# Convert a local Markdown file into a formatted Google Doc
./drivectl docs create "Architecture Design Doc" ./docs/DESIGN_PHILOSOPHY.md -O json
```

#### List Google Doc tabs

```bash
# View the hierarchy of tabs and child tabs in a Document
./drivectl docs tabs <document-id> -O json
```

### Google Sheets (`sheets`)

```bash
# Export a sheet tab as CSV to stdout
./drivectl sheets get <spreadsheet-id> --sheet "Sheet1"

# Read a specific A1 range
./drivectl sheets get-range <spreadsheet-id> --sheet "Sheet1" --range "A1:C5" -O json

# Update a cell or range
./drivectl sheets update-range <spreadsheet-id> "Approved" --sheet "Sheet1" --range "B2" -O json
```

### Dynamic Discovery Calls (`call`)

Use `drivectl call` to invoke any Google API endpoint described by Google's Discovery Service. Key names in `--payload` are automatically routed to path parameters, query parameters, or the JSON request body based on the Discovery schema:

```bash
# List files via drive.v3.files.list
./drivectl call drive.v3.files.list --payload '{"pageSize": 5}' -O json

# Fetch specific fields for a known file
./drivectl call drive.v3.files.get --payload '{"fileId": "YOUR_FILE_ID", "fields": "name,mimeType"}' -O json

# Create a new Google Doc via docs.v1.documents.create
./drivectl call docs.v1.documents.create --payload '{"title": "My New Document"}' -O json
```

### Automating Workflows (`run`)

Chain multiple `drivectl` commands in a declarative JSON recipe ([`recipes/sample.json`](recipes/sample.json)):

```bash
./drivectl run recipes/sample.json
```

```json
{
  "name": "Quick Diagnostics",
  "description": "Lists recent documents.",
  "steps": [
    ["list", "--limit", "2"]
  ]
}
```

## MCP Server Mode

`drivectl` includes a built-in Model Context Protocol (MCP) server (`github.com/modelcontextprotocol/go-sdk`) that exposes both native tools (`drivectl_list`, `drivectl_read`, `drivectl_call`) and dynamically generated Discovery tools (`drive_v3_files_list`, `docs_v1_documents_get`, `sheets_v4_spreadsheets_get`).

```bash
# Start MCP server over stdio (default for AI coding agents)
./drivectl --mcp

# Start MCP server over HTTP (Streamable HTTP + SSE compatibility)
./drivectl --mcp-http :8080
```

## Agent Skill & Gemini CLI Configuration

- **Bundled Agent Skill:** [`skills/google-drive-cli/SKILL.md`](skills/google-drive-cli/SKILL.md) provides progressive-disclosure instructions and references ([`drive.md`](skills/google-drive-cli/references/drive.md), [`docs.md`](skills/google-drive-cli/references/docs.md), [`sheets.md`](skills/google-drive-cli/references/sheets.md), [`discovery.md`](skills/google-drive-cli/references/discovery.md)) for AI coding agents.
- **Gemini CLI:** Copy [`settings.json.sample`](settings.json.sample) to `.gemini/settings.json` in your project root and update the binary path to register `drivectl` as an MCP server.

## Local Development

### Prerequisites

- **Go 1.26+** (pinned to `go 1.26.11` in [`go.mod`](go.mod))
- **golangci-lint v2** (optional, for complexity and static analysis linting)
- **Node.js 22+** (optional, for building the Astro Starlight documentation site in `site/`)

### Build, Test, and Lint

```bash
# Clone the repository
git clone https://github.com/ghchinoy/drivectl.git
cd drivectl

# Build the CLI binary
go build -o drivectl .

# Run unit and in-memory MCP integration tests with the race detector
go test -p 2 -race ./...

# Run linters and complexity checks (.golangci.yml)
golangci-lint run -E gocyclo -E cyclop -E gocognit ./...

# Check for known vulnerabilities
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

### Documentation Site (`site/`)

```bash
cd site
npm install
ASTRO_TELEMETRY_DISABLED=1 npm run check
ASTRO_TELEMETRY_DISABLED=1 npm run build
npm run check:docs
```

## Releasing & Publishing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions ([`.github/workflows/release.yaml`](.github/workflows/release.yaml)). Pushing a semantic version tag builds cross-platform archives (`Darwin` and `Linux` on `x86_64` and `arm64`) and publishes SHA-256 checksums to GitHub Releases:

```bash
git checkout main && git pull
git tag v0.2.0
git push origin v0.2.0
```

To test a release build locally without publishing:

```bash
goreleaser release --snapshot --clean
```

See [`docs/RELEASING.md`](docs/RELEASING.md) for the full release checklist.

## Contributing

Issues and pull requests are welcome! Before opening a pull request:

1. Read [`docs/DESIGN_PHILOSOPHY.md`](docs/DESIGN_PHILOSOPHY.md) to decide whether a feature belongs in the generic `call` command (standard JSON REST endpoints) or a dedicated native subcommand (streaming media, multi-step protocols, or AST conversion).
2. Ensure `go test -p 2 -race ./...` and `golangci-lint run -E gocyclo -E cyclop -E gocognit ./...` pass with zero issues.
3. For larger changes, open an issue first to discuss the design.

## Troubleshooting

- **`authentication required: credentials file not found`:** Run `drivectl auth login --secret-file /path/to/client_secret.json` once to initialize `~/.config/drivectl/credentials.json` and `token.json`.
- **Browser does not open on a remote or SSH host:** Pass `--no-browser-auth` to `drivectl auth login` and open the printed authorization URL on your local machine.
- **`403 Forbidden: Google Docs/Sheets API has not been used in project`:** Enable the corresponding API (`drive.googleapis.com`, `docs.googleapis.com`, `sheets.googleapis.com`) in the Google Cloud Console for the project that owns your OAuth client ID.

## License

Licensed under the [Apache License 2.0](LICENSE).

## Disclaimer

> [!CAUTION]
> This is **not** an officially supported Google product.
