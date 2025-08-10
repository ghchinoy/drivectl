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
11. Place this file in a secure location. By default, `drivectl` looks for it at `~/secrets/client_google-drive-api_ghchinoy-genai-blackbelt-fishfooding.json`, but you can specify a different path using the `--secret-file` flag.

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
