# drivectl

A command-line tool for interacting with the Google Drive API.

## Features

*   List files with powerful query capabilities.
*   Describe file metadata.
*   Download files, with special handling for Google Docs (converts to plain text).
*   Simple and secure OAuth 2.0 authentication flow.

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
5.  Go to **APIs & Services > Credentials**.
6.  Click **Create Credentials > OAuth client ID**.
7.  Select **Desktop app** as the application type.
8.  Give it a name (e.g., "drivectl").
9.  Click **Create**. A window will appear with your client ID and client secret. Click **Download JSON**.
10. Rename the downloaded file to `client_secret.json`.
11. Place this file in a secure location. You will need to provide the path to this file using the `--secret-file` flag or by setting the `DRIVE_SECRETS` environment variable.

## Usage

### First-time Authentication

The first time you run any command, `drivectl` will open a browser window for you to authorize access to your Google Drive account. After you grant access, your browser will be redirected to a page that says "Authentication successful!". You can then close the browser window. Your authentication token will be stored securely for future use.

If you are on a system without a graphical browser, you can use the `--no-browser-auth` flag. This will print a URL to the console, which you can open on another machine. You will then need to copy the authorization code from the browser and paste it back into the terminal.

### Commands

**List files**

```bash
# List the first 100 files
./drivectl list

# List up to 20 files
./drivectl list --limit 20

# List all Google Docs
./drivectl list -q "mimeType='application/vnd.google-apps.document'"

# List all files containing "MyProject" in the name
./drivectl list -q "name contains 'MyProject'"
```

The `--query` (`-q`) flag uses the Google Drive API's query language. You can build powerful and specific queries. For more details on the query syntax, see the [official Google Drive documentation](https://developers.google.com/drive/api/v3/search-files).

**Describe a file**

```bash
# Get detailed metadata for a file
./drivectl describe <file-id>
```

**List tabs in a document**

```bash
# List the tabs by their index number
./drivectl tabs <document-id>
```

**Get a file or tab content**

```bash
# Get the content of a file and print it to the console
./drivectl get <file-id>

# Get a Google Doc and save it as a text file (default format)
./drivectl get <google-doc-id> -o my-document.txt

# Export a Google Doc as Markdown
./drivectl get <google-doc-id> --format md -o my-document.md

# Export a Google Doc as a PDF
./drivectl get <google-doc-id> --format pdf -o my-document.pdf

# Get the content of a specific tab (e.g., the first tab)
./drivectl get <google-doc-id> --tab-index 0

# Download a regular file (e.g., a PDF)
./drivectl get <pdf-file-id> -o my-file.pdf
```

### Google Sheets and Docs

**List sheets in a spreadsheet**

```bash
./drivectl sheets list <spreadsheet-id>
```

**Get a sheet as CSV**

```bash
./drivectl sheets get <spreadsheet-id> --sheet <sheet-name>
```

**Get a specific range from a sheet**

```bash
./drivectl sheets get-range <spreadsheet-id> --sheet <sheet-name> --range <A1-notation>
```

**Update a specific range in a sheet**

```bash
./drivectl sheets update-range <spreadsheet-id> <value> --sheet <sheet-name> --range <A1-notation>
```

**List tabs in a document**

```bash
./drivectl docs tabs <document-id>
```

## MCP Server Mode

`drivectl` can also be run as an MCP server, exposing its commands as tools that can be called by an MCP client.

### Starting the Server

You can start the MCP server in two modes:

**Stdio Mode:**

```bash
./drivectl --mcp
```

This will start the server on the standard input/output.

**HTTP Mode:**

```bash
./drivectl --mcp-http :8080
```

This will start the server on port 8080.

### Interacting with the Server

You can use the `mcptools` CLI to interact with the server.

**List available tools:**

```bash
mcptools tools ./drivectl --mcp
```

**Call a tool:**

```bash
# List files
mcptools call list ./drivectl --mcp

# Get a file
mcptools call get -p '{"file-id": "<your-file-id>"}' ./drivectl --mcp
```

## Gemini CLI Configuration

This repository contains a sample `settings.json.sample` file. To use this tool with the Gemini CLI, you should copy this file to `.gemini/settings.json` in your project root and replace the placeholder values with your actual credentials.

The `.gemini/` directory and its contents are ignored by git.

# Disclaimer

This is not an officially supported Google product.