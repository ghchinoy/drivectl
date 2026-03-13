# drivectl

An experimental command-line tool for interacting with the Google Drive API.

Please review either the [Gemini CLI Workspace Extension](https://github.com/gemini-cli-extensions/workspace) or the [Workspace CLI](https://github.com/googleworkspace/cli) for more up-to-date examples of interacting with Workspace.

## Features

*   **Robust Authentication:** Secure OAuth 2.0 login with automatic local caching and silent token refreshing.
*   **Agent-Friendly UX:** Thoughtful CLI design with semantic command grouping, color-coded output, proactive error hints, and deterministic JSON formatting (`-O json`) to simplify LLM integration and scripting.
*   **Dynamic API Capabilities:** A generic `call` subcommand powered by Google API Discovery Documents to hit *any* Google Workspace endpoint dynamically.
*   **Google Drive Integration:** List files with powerful query capabilities, describe file metadata, and download files.
*   **Google Docs Integration:** Convert Markdown files into richly formatted Google Docs, export Docs back to raw Markdown (parsing the AST), PDF, or plain text, and manage document tabs.
*   **Google Sheets Integration:** Export sheets to CSV, read explicit cell ranges via A1 notation, and update cell values.
*   **Composable Recipes:** Execute sequences of CLI commands defined in JSON files via `drivectl run` for complex, automated workflows.
*   **MCP Server Mode:** Run `drivectl` as an MCP server to expose Workspace interactions directly to LLM agents using Discovery-driven schemas.

## Installation

1.  Ensure you have Go installed on your system.
2.  Clone this repository.
3.  Build the tool:
    ```bash
    go build -o drivectl .
    ```
4.  (Optional) Move the `drivectl` executable to a directory in your `PATH` (e.g., `/usr/local/bin`).

## Getting a Google Drive API Client Secret

1.  Go to the [Google Cloud Console](https://console.cloud.google.com/).
2.  Create a new project or select an existing one.
3.  In the navigation menu, go to **APIs & Services > Library**.
4.  Search for and enable the following APIs:
    *   **Google Drive API**
    *   **Google Docs API**
    *   **Google Sheets API**
5.  Go to **APIs & Services > Credentials**.
6.  Click **Create Credentials > OAuth client ID**.
7.  Select **Desktop app** as the application type.
8.  Give it a name (e.g., "drivectl").
9.  Click **Create**. A window will appear with your client ID and client secret. Click **Download JSON**.
10. Rename the downloaded file to `client_secret.json` and keep it handy for your first login.

## Usage

### First-time Authentication

`drivectl` handles authentication robustly. You only need to provide your client secret file once. The CLI will cache the secrets and your authorization tokens locally in `~/.config/drivectl/`, automatically refreshing your session in the background as needed.

To authenticate for the first time, run:

```bash
./drivectl auth login --secret-file /path/to/your/client_secret.json
```

This will automatically open your default browser. Once you grant permissions, you can return to your terminal. All subsequent commands can be run without passing the `--secret-file` flag!

*(If you are on a headless system, you can use the `--no-browser-auth` flag to print a manual authorization URL).*

### Dynamic Discovery Calls

You can dynamically execute *any* Google API endpoint using the `call` subcommand. It fetches the latest Google Discovery schema to build the request.

```bash
# List files using the generic discovery caller
./drivectl call drive.v3.files.list --payload '{"pageSize": 5}'

# Fetch specific fields for a known file
./drivectl call drive.v3.files.get --payload '{"fileId": "YOUR_FILE_ID", "fields": "name,mimeType"}'

# Create a new Google Doc
./drivectl call docs.v1.documents.create --payload '{"title": "My New Document"}'
```

### Core Commands

**List files**

```bash
# List the first 100 files
./drivectl list

# List up to 20 files
./drivectl list --limit 20

# List all Google Docs using Drive query syntax
./drivectl list -q "mimeType='application/vnd.google-apps.document'"
```

**Get file content**

```bash
# Download a file to stdout
./drivectl get <file-id>

# Export a Google Doc as Markdown
./drivectl get <google-doc-id> --format md -o my-document.md
```

### Google Docs & Sheets

**Create a Google Doc from Markdown**

```bash
# Converts the markdown file into a formatted Google Doc
./drivectl docs create "My New Design Doc" ./docs/design.md
```

**List Google Doc Tabs**

```bash
# View the structural hierarchy of tabs in a Doc
./drivectl docs tabs <document-id>
```

**Interact with Google Sheets**

```bash
# Export a sheet as a CSV
./drivectl sheets get <spreadsheet-id> --sheet "Sheet1"

# Get values using A1 notation
./drivectl sheets get-range <spreadsheet-id> --sheet "Sheet1" --range "A1:C5"

# Update a specific cell
./drivectl sheets update-range <spreadsheet-id> "New Value" --sheet "Sheet1" --range "B2"
```

### Advanced Workflows

**Deterministic JSON Output**

All commands support a `-O json` flag to bypass terminal UI formatting and emit pure, parseable JSON for shell pipelines or AI agents.

```bash
# Get structured JSON output of Drive files
./drivectl list -q "name contains 'Project'" -O json
```

**Executing CLI Recipes**

You can string multiple `drivectl` commands together using a JSON recipe file to automate redundant tasks.

```bash
# Run the sequential steps defined in a recipe
./drivectl run recipes/sample.json
```

*Example Recipe (`recipes/sample.json`):*
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

`drivectl` can also be run as an MCP server, exposing its commands as tools that can be called by an MCP client (like Gemini).

### Starting the Server

**Stdio Mode (Default for Agents):**

```bash
./drivectl --mcp
```

**HTTP Mode:**

```bash
./drivectl --mcp-http :8080
```

## Gemini CLI Configuration

This repository contains a sample `settings.json.sample` file. To use this tool with the Gemini CLI, you should copy this file to `.gemini/settings.json` in your project root and replace the placeholder values with your actual configuration.

# License

Apache-2.0

# Disclaimer

> [!CAUTION]
> This is **not** an officially supported Google product.