# drivectl

[![Go Version](https://img.shields.io/github/go-mod/go-version/ghchinoy/drivectl)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/ghchinoy/drivectl)](https://github.com/ghchinoy/drivectl/releases)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

A Go command-line interface (CLI) and Model Context Protocol (MCP) server for interacting with **Google Drive**, **Google Docs**, **Google Sheets**, and **Google Slides**, plus dynamic calls to any Google Workspace REST endpoint via Google API Discovery Documents. Built for both interactive terminal workflows and AI coding agents with deterministic JSON output (`-O json`), Markdown-to-Docs AST conversion, and multi-step JSON recipes.

> [!IMPORTANT]
> **Official Google Workspace CLI (`gwscli`) & Remote MCP Servers Available**
>
> Google now provides an official [Google Workspace CLI (`gwscli`)](https://developers.google.com/workspace/guides/workspace-cli) (prebuilt Go CLI backed by `*mcp.googleapis.com` for Drive, Docs, Sheets, Slides, Gmail, Calendar, Chat, Tasks, and People) and hosted [Google Workspace Remote MCP servers](https://developers.google.com/workspace/guides/configure-mcp-servers#support-products) (10 endpoints adding Classroom and Universal Search). Google also publishes a separate open-source Rust CLI, [`gws`](https://github.com/googleworkspace/cli), built directly over Discovery APIs.
>
> **Start with the official tools** for general Workspace automation, hosted MCP integrations, or multi-product workflows (Gmail, Calendar, Chat, Tasks, Universal Search).
>
> **`drivectl` remains useful when you need:**
> - An **embeddable Go library** (`pkg/drive`, `pkg/discovery`, `mcp.RegisterTools`) for stateless Cloud Run backends or custom Go MCP hosts
> - **Stable, versioned CLI flags for static scripts & CI** (`gwscli` discovers schemas dynamically and explicitly warns against relying on fixed signatures in static scripts)
> - **Markdown $\leftrightarrow$ Google Docs** AST conversion (`docs create`, `get --format md|pdf`) and `docs tabs` hierarchy inspection
> - **Google Slides CLI ergonomics** (`slides get`, `inspect`, `replace`, `notes`, plus in-place `.pptx` $\rightarrow$ Slides conversion via `upload --convert slides --file-id`)
> - **Drive revisions, RFC 4180 Sheets CSV export, zero usage telemetry**, or a **single-binary local MCP server** without Workspace Developer Preview enrollment

<details>
<summary><strong>Quick comparison: drivectl vs. official <code>gwscli</code>, Remote MCP &amp; open-source <code>gws</code></strong></summary>

| Capability | `drivectl` | Official CLI (`gwscli`) | Official Remote MCP (`*mcp.googleapis.com`) | Open-Source CLI (`gws`) |
|---|---|---|---|---|
| **Runtime** | Open-source Go binary (CLI + local stdio/HTTP MCP) | Closed-source Go binary over `*mcp.googleapis.com` | Google-hosted remote Streamable HTTP | Open-source Rust binary + local MCP mode |
| **Product breadth** | Drive, Docs, Sheets, Slides (+ `call` Discovery) | 9 services: Drive, Docs, Sheets, Slides, Gmail, Calendar, Chat, Tasks, People *(Preview)* | 10 endpoints incl. Classroom & Universal Search (`workspacemcp`) | All Workspace Discovery APIs + `+workflow` helpers |
| **Embeddable Go SDK** | Yes (`pkg/drive`, `pkg/discovery`, `mcp`) | No | No | No |
| **Static-script stability** | Yes (stable versioned flags & JSON recipes) | No (runtime discovery; docs warn against static scripts) | Dynamic MCP tool discovery | Dynamic Discovery commands |
| **Docs Markdown AST & PDF** | Yes (`docs create`, `get --format md\|pdf`, `docs tabs`) | Raw JSON (`read_doc`, `update_doc`) | Raw JSON (`read_doc`, `update_doc`) | Plain-text append (`docs +write`) |
| **Slides CLI & `.pptx` convert** | Yes (`create`, `get` MD, `inspect`, `replace`, `notes`, `--convert slides`) | Raw JSON + `read_slide_page_thumbnail` | Raw JSON + `read_slide_page_thumbnail` | Raw Discovery calls |
| **Multimodal extraction** | Native Docs/Sheets/Slides only | Yes (`read_file_content` text/MD + comments) | Yes (`read_file_content` parses PDF, Office, PNG/JPEG) | Raw Discovery calls |
| **Auth, Security & Telemetry** | Desktop OAuth (PKCE + AES-256-GCM, `--readonly`/`--services`, `--agentic` profiles, `DRIVECTL_TOKEN`); no preview gate; **zero telemetry** | Personal Mode onboarding, incremental/`--bundle=readonly` scopes, HITL elicitation, Model Armor; **telemetry on by default** (opt-out) | Developer Preview + `is_agentic` OAuth flag, Workspace DLP/governance, HITL elicitation | OS keychain AES-256-GCM, Service Accounts, Model Armor |

</details>

> [!NOTE]
> **Sister CLI (`share-with`):** For dedicated Google Drive ACL and sharing workflows, use [`share-with`](https://github.com/ghchinoy/share-with) ([docs](https://ghchinoy.github.io/share-with/)), which enforces silent-only permission grants (`sendNotificationEmail=false` at the HTTP transport layer), `?resourcekey=...` URL parsing, and pre-grant no-downgrade collaborator protection. Also see the [Gemini CLI Workspace Extension](https://github.com/gemini-cli-extensions/workspace).

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
- [Using as a Go Library (`pkg/drive`, `pkg/discovery`, `mcp`)](#using-as-a-go-library-pkgdrive-pkgdiscovery-mcp)
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
- **Dual-Mode MCP Server (`--mcp` / `--mcp-http`):** Expose 14 Discovery-driven Google Workspace tools (`drive.v3.files.*`, `docs.v1.documents.*`, `sheets.v4.spreadsheets.*`, `slides.v1.presentations.*`) and embedded query/A1 cheat-sheet resources over stdio or hardened loopback HTTP.

## Installation

### Homebrew (macOS & Linux)

```bash
brew install ghchinoy/tap/drivectl
```

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

`drivectl` authenticates as your Google user via an **OAuth 2.0 Desktop app** client secret.

> [!IMPORTANT]
> You need an **OAuth 2.0 Client ID (Desktop app)** JSON file whose top-level key is `{"installed": {...}}`. Do **not** use a **Service Account** JSON key (`{"type": "service_account", ...}` — not currently supported), an **API key**, or a **Web application** OAuth client (`{"web": ...}`, which does not allow ephemeral loopback redirect ports `http://127.0.0.1:<port>`).

### Quick Setup: Google Personal Mode (Single User)

For personal single-user setups, Google Cloud Console's **Personal mode onboarding** creates a Desktop OAuth 2.0 client tied to your Google Account, enables core Workspace APIs automatically, and skips OAuth consent screen setup and app verification:

1. Open [Personal mode setup (Desktop app)](https://console.cloud.google.com/auth/overview/personal?apis=ws&clientType=desktop) (or [Personal mode setup for AI agent / MCP](https://console.cloud.google.com/auth/overview/personal?apis=ws&clientType=desktop&isAgent=true) for a dedicated `--agentic` profile).
2. Select or create a personal-mode project, click **Create Client**, and download the JSON as `client_secret.json`.
3. Authenticate with `drivectl auth login --secret-file client_secret.json` (or `--agentic` for an agent client).

### Manual Setup (Workspace Org or Existing Project)

1. Open the [Google Cloud Console](https://console.cloud.google.com/) and create or select a project.
2. Navigate to **APIs & Services > Library** and enable:
   - **Google Drive API** (`drive.googleapis.com`)
   - **Google Docs API** (`docs.googleapis.com`)
   - **Google Sheets API** (`sheets.googleapis.com`)
   - **Google Slides API** (`slides.googleapis.com`)
3. Configure the OAuth consent screen under **Google Auth Platform** (or **APIs & Services > OAuth consent screen**):
   - Under **Branding**, set an app name and support email.
   - Under **Audience**, choose **Internal** (if you are in a Google Workspace organization and only org users will sign in) or **External** (for personal Gmail accounts or cross-org use).
   - If **External** is left in **Testing** publishing status, open **Audience > Test users** and add the Google email address(es) you will authenticate with — otherwise login fails with `Error 403: access_denied`. *(Note: Google expires refresh tokens for External apps in Testing mode after 7 days; re-run `drivectl auth login` when needed.)*
4. Navigate to **Google Auth Platform > Clients** (or **APIs & Services > Credentials**), click **Create Client** / **Create Credentials > OAuth client ID**, and select **Desktop app** as the application type.
   - If creating a dedicated client for AI agent / MCP usage, check **Use this client for an AI-powered agent** and authenticate it with `drivectl auth login --agentic --secret-file <path>`.
5. Download the JSON file (verify it starts with `{"installed": ...}`), save it as `client_secret.json`, and pass it once to `drivectl auth login --secret-file client_secret.json` (or set `DRIVE_SECRETS=/path/to/client_secret.json`).

## Usage & Command Reference

| Command | Group | Purpose |
|---|---|---|
| `drivectl auth login` | `auth` | Authenticate with Google OAuth 2.0 (PKCE, encrypted cache, `--readonly`, `--services`, `--agentic`, `--prompt-consent`) |
| `drivectl auth status` | `auth` | Inspect active profile, agentic flag, bearer token, scopes, encryption status, and token expiry |
| `drivectl auth refresh` | `auth` | Force an immediate OAuth token refresh and persist the updated encrypted token |
| `drivectl auth logout` | `auth` | Revoke the active OAuth token at Google and remove the local cached token (`--local-only` skips remote revocation) |
| `drivectl auth reset` | `auth` | Revoke tokens and delete all cached credentials, metadata, and key files for a profile |
| `drivectl auth scopes [cmd...]` | `auth` | Print minimal OAuth scopes required for a command (or pass `--show-scopes` to any command) |
| `drivectl list` | `workspace` | List and search files in Google Drive |
| `drivectl describe <file-id>` | `workspace` | Fetch detailed metadata for a Drive file |
| `drivectl get <file-id>` | `workspace` | Download a file or export a Google Doc (`raw`, `txt`, `md`, `pdf`) |
| `drivectl upload <local-path>` | `workspace` | Upload a local file (with optional `--file-id` in-place update or `--convert slides\|docs\|sheets`) |
| `drivectl revisions <file-id>` | `workspace` | List the revision history of a file |
| `drivectl comments <file-id>` | `workspace` | List threaded comments and replies on a file |
| `drivectl docs create <title> [md]` | `workspace` | Create a Google Doc (optionally converting a local Markdown file) |
| `drivectl docs tabs <doc-id>` | `workspace` | Inspect the structural hierarchy of tabs in a Google Doc |
| `drivectl sheets get <id>` | `workspace` | Export a Google Sheet tab to CSV |
| `drivectl sheets get-range <id>` | `workspace` | Read cell values from an A1 range |
| `drivectl sheets update-range <id> <val>` | `workspace` | Write a value to an A1 range |
| `drivectl slides get <pres-id>` | `workspace` | Render a Google Slides presentation (slides, tables, notes) as Markdown or JSON |
| `drivectl slides create <title>` | `workspace` | Create a new Google Slides presentation via the Slides API |
| `drivectl slides inspect <pres-id>` | `workspace` | List slide IDs, element objectIds, text previews, and speaker notes IDs |
| `drivectl slides replace <id> <old> <new>` | `workspace` | Replace all matching text occurrences across a presentation |
| `drivectl slides notes <id> --slide N` | `workspace` | Read or update (`--set` / `--file`) speaker notes on a slide |
| `drivectl call <api.ver.res.method>` | `api` | Dynamically invoke any Google REST endpoint via Discovery |
| `drivectl run <recipe.json>` | `api` | Execute a sequence of `drivectl` commands from a JSON recipe |
| `drivectl --mcp` / `--mcp-http :8080` | root flags | Run `drivectl` as an MCP server over stdio or HTTP |

### Authentication (`auth`)

`drivectl` caches your client secret (`client_secret.json`) and AES-256-GCM encrypted OAuth token (`token.json`) in `~/.config/drivectl/` (or `~/.config/drivectl/profiles/<name>/` for named profiles) and persists refreshed tokens automatically. You can supply the secret path via `--secret-file` or `DRIVE_SECRETS`, select a profile via `--profile` or `DRIVECTL_PROFILE`, pass a pre-obtained bearer token in CI/headless environments via `DRIVECTL_TOKEN` or `--token-file` (or `-` for stdin), and optionally supply your own 32-byte base64 key via `DRIVECTL_ENCRYPTION_KEY`.

```bash
# Browser-based OAuth login (PKCE S256 + CSRF state on an ephemeral 127.0.0.1 loopback port)
./drivectl auth login --secret-file /path/to/client_secret.json

# Least-privilege read-only login for specific services (subsequent --services logins merge incrementally)
./drivectl auth login --readonly --services drive,docs --secret-file /path/to/client_secret.json

# Inspect minimal required scopes for any command before calling it
./drivectl auth scopes upload -O json
./drivectl list --show-scopes

# Dedicated agentic OAuth profile (for clients marked "Use this client for an AI-powered agent")
./drivectl auth login --agentic --secret-file /path/to/agent_client_secret.json

# Inspect status, force refresh, revoke & log out, or wipe a profile
./drivectl auth status --profile agent -O json
./drivectl auth refresh
./drivectl auth logout --profile agent
./drivectl auth reset --profile agent

# Headless / CI execution with a pre-obtained OAuth access token (no local client_secret.json required)
DRIVECTL_TOKEN="$ACCESS_TOKEN" ./drivectl list --limit 5 -O json

# Force consent screen to ensure a fresh refresh token
./drivectl auth login --prompt-consent

# Headless / remote session login (prints manual authorization URL)
./drivectl auth login --secret-file /path/to/client_secret.json --no-browser-auth
```

> [!NOTE]
> - **Ephemeral loopback callback:** Default browser login binds `127.0.0.1:0` with PKCE (`S256`) and random CSRF `state` verification—no fixed port conflicts with `:8080` or local dev servers.
> - **Least-privilege & incremental scopes:** Pass `--readonly` and/or `--services drive,docs,sheets,slides` to limit granted scopes (`--replace-scopes` replaces prior profile scopes instead of merging; `--print-scopes` previews resolved scopes without opening a browser).
> - **Pre-obtained tokens (`DRIVECTL_TOKEN` / `--token-file`):** Works in both CLI and MCP modes without local disk state. By design, `drivectl` avoids a raw `--token` CLI flag so access tokens do not leak into `ps` or `/proc/<pid>/cmdline`.
> - **Agentic OAuth profiles (`--agentic`):** When running `drivectl --mcp` or `--mcp-http`, `drivectl` automatically prefers the `agent` profile (or `DRIVECTL_AGENT_SECRETS`) when configured, keeping AI agent credentials separate from interactive CLI credentials.
> - **Encrypted token storage:** Cached tokens are encrypted at rest with AES-256-GCM (`.encryption_key` with `0600` permissions, or `DRIVECTL_ENCRYPTION_KEY`). Legacy plaintext `token.json` files are read seamlessly and upgraded in place.
> - **Headless (`--no-browser-auth`):** Open the printed URL in a browser on any device and grant access. Copy the `code` query parameter value from the redirect URL and paste it into the terminal prompt.

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

# Upload a local .pptx and convert it directly into a native Google Slides presentation
./drivectl upload ./deck.pptx --name "DevFest 2026 Workshop" --convert slides

# Update an existing Google Slides deck in place (preserves URL and sharing permissions)
./drivectl upload ./deck.pptx --file-id <existing-id> --convert slides
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

### Google Slides (`slides`)

```bash
# Create a new Google Slides presentation
./drivectl slides create "DevFest 2026 Workshop" -O json

# Render slides, tables, and speaker notes as Markdown
./drivectl slides get <presentation-id>

# Inspect slide IDs, element objectIds, text previews, and speakerNotesObjectId
./drivectl slides inspect <presentation-id> --slide 3

# Replace text across all slides
./drivectl slides replace <presentation-id> "Speaker Name" "G. Hussain Chinoy" --match-case

# Read or update speaker notes on a specific slide
./drivectl slides notes <presentation-id> --slide 1
./drivectl slides notes <presentation-id> --slide 1 --set "Welcome everyone!"

# Or upload a local .pptx and convert directly into native Google Slides (or update in place)
./drivectl upload ./deck.pptx --name "DevFest 2026 Workshop" --convert slides
./drivectl upload ./deck.pptx --file-id <existing-id> --convert slides
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

# Pass a large JSON body from a file or stdin (ideal for batchUpdate requests)
./drivectl call slides.v1.presentations.batchUpdate --payload-file ./batch.json
cat ./batch.json | ./drivectl call slides.v1.presentations.batchUpdate --payload-file -
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

> [!NOTE]
> Google now hosts official remote MCP servers for [Drive, Docs, Sheets, Slides, Gmail, Calendar, Chat, People, Classroom, and Universal Search](https://developers.google.com/workspace/guides/configure-mcp-servers#support-products). Prefer those managed endpoints when connecting AI agents directly to Workspace; use `drivectl --mcp` / `mcp.RegisterTools` when you need a local single-binary MCP server without Developer Preview enrollment or want to mount Discovery-backed tools inside a custom Go MCP host.

`drivectl` includes a built-in Model Context Protocol (MCP) server (`github.com/modelcontextprotocol/go-sdk`) that registers 14 dynamically generated Discovery tools (`drive.v3.files.{list,get,create,update}`, `docs.v1.documents.{create,get,batchUpdate}`, `sheets.v4.spreadsheets.{create,get}`, `sheets.v4.spreadsheets.values.{get,update}`, `slides.v1.presentations.{create,get,batchUpdate}`) and 2 embedded cheat-sheet resources (`embedded:drive-query-cheat-sheet`, `embedded:a1-notation-cheat-sheet`).

```bash
# Start MCP server over stdio (default for AI coding agents; auto-uses "agent" profile if configured)
./drivectl --mcp

# Start MCP server over HTTP (defaults ":8080" to 127.0.0.1:8080 with Origin validation)
./drivectl --mcp-http :8080

# Require Bearer token authentication on HTTP MCP requests
DRIVECTL_MCP_AUTH_TOKEN="your-secret-token" ./drivectl --mcp-http 127.0.0.1:8080
```

## Using as a Go Library (`pkg/drive`, `pkg/discovery`, `mcp`)

External Go services (such as Cloud Run backends or MCP hosts) can import `github.com/ghchinoy/drivectl` directly with stateless OAuth bearer tokens or `oauth2.TokenSource` instances—without local disk state or stdout output:

```go
import (
	"context"
	"net/http"

	drivectlmcp "github.com/ghchinoy/drivectl/mcp"
	"github.com/ghchinoy/drivectl/pkg/discovery"
	"github.com/ghchinoy/drivectl/pkg/drive"
)

func example(ctx context.Context, accessToken string) error {
	// 1. Construct a stateless HTTP client and Workspace services bundle
	client := drive.NewClientFromBearerToken(ctx, accessToken)
	svcs, err := drive.NewServicesFromClient(ctx, client)
	if err != nil {
		return err
	}

	// 2. Fetch Google Docs, Sheets (RFC 4180 CSV), or Slides (Markdown)
	csvData, _ := drive.FetchSheetAsCSV(svcs.Sheets, "spreadsheet-id", "Sheet1")
	slidesMD, _ := drive.FetchPresentationAsMarkdown(svcs.Slides, "presentation-id")
	_, _ = csvData, slidesMD

	// 3. Call any Discovery endpoint directly
	_, err = discovery.CallMethod(ctx, client, "drive.v3.files.get", map[string]any{
		"fileId": "file-id",
		"fields": "id,name,mimeType",
	})
	return err
}
```

To mount `drivectl`'s resources and Discovery tools onto an existing `*mcp.Server`:

```go
err := drivectlmcp.RegisterTools(ctx, server, func(ctx context.Context) (*http.Client, error) {
	return drive.NewClientFromBearerToken(ctx, accessToken), nil
})
```

## Agent Skill & Gemini CLI Configuration

- **Bundled Agent Skill:** [`skills/google-drive-cli/SKILL.md`](skills/google-drive-cli/SKILL.md) provides progressive-disclosure instructions and references ([`drive.md`](skills/google-drive-cli/references/drive.md), [`docs.md`](skills/google-drive-cli/references/docs.md), [`sheets.md`](skills/google-drive-cli/references/sheets.md), [`slides.md`](skills/google-drive-cli/references/slides.md), [`discovery.md`](skills/google-drive-cli/references/discovery.md)) for AI coding agents.
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

- **`client secret file not found` / `unable to read client secret file`:** Run `drivectl auth login --secret-file /path/to/client_secret.json` (or set `DRIVE_SECRETS=/path/to/client_secret.json`) once to initialize `~/.config/drivectl/client_secret.json` and `~/.config/drivectl/token.json`.
- **`unable to parse client secret file to config`:** Ensure your JSON file is an **OAuth 2.0 Desktop app** client secret (`{"installed": {...}}`), not a Service Account key (`{"type": "service_account", ...}`) or API key.
- **`Error 403: access_denied` during browser consent:** Your Google Cloud project's OAuth consent screen is set to **External** in **Testing** mode. Open **Google Auth Platform > Audience > Test users** in the Google Cloud Console and add the Google account you are signing in with.
- **`decrypting cached token` / rotated encryption key:** If `.encryption_key` or `DRIVECTL_ENCRYPTION_KEY` changed, clear the cached token with `drivectl auth logout` and re-authenticate with `drivectl auth login`.
- **Repeated hourly sign-in prompts (`has_refresh_token: false` in `drivectl auth status -O json`):** Google only issues a refresh token on first offline consent. Remove the app's existing grant in [Google Account connections](https://myaccount.google.com/connections) and re-run `drivectl auth login` to receive a fresh refresh token.
- **Browser does not open on a remote or SSH host:** Pass `--no-browser-auth` to `drivectl auth login`, open the printed authorization URL on your local machine, and paste the `code` query parameter from the resulting redirect URL into the terminal.
- **`403 Forbidden: Google Drive/Docs/Sheets/Slides API has not been used in project`:** Enable the corresponding API (`drive.googleapis.com`, `docs.googleapis.com`, `sheets.googleapis.com`, `slides.googleapis.com`) in the Google Cloud Console for the project that owns your OAuth client ID.

## License

Licensed under the [Apache License 2.0](LICENSE).

## Disclaimer

> [!CAUTION]
> This is **not** an officially supported Google product.
