# Implementation Plan: drivectl

This document outlines the steps to create the `drivectl` Go command-line tool. We will follow this plan in a staged manner: implement, test, check off, and update as we go.

## Phase 1: Project Foundation

### Project Setup & Dependencies
- [x] Initialize Go module (`go mod init github.com/user/drivectl`). (Note: `go.mod` already exists, will verify/update).
- [x] Add core dependencies using `go get`:
  - [x] `github.com/spf13/cobra`
  - [x] `golang.org/x/oauth2`
  - [x] `google.golang.org/api/drive/v3`
  - [x] `github.com/jaytaylor/html2text`
- [x] Add new dependency for opening the browser:
  - [x] `github.com/pkg/browser`

### Authentication
- [x] Create a new `auth` package (`cmd/auth.go`). (Note: Kept in `cmd` package for simplicity).
- [x] Implement OAuth 2.0 flow to retrieve and cache `token.json`.
- [x] The client secret path will be configurable via a persistent flag on the root command (`--secret-file`), defaulting to `~/secrets/client_google-drive-api_ghchinoy-genai-blackbelt-fishfooding.json`.
- [x] The token will be stored in `~/.config/drivectl/token.json`.

#### Enhanced Authentication Flow (Local Webserver)
- [x] Add a `--no-browser-auth` flag to the root command.
- [x] If `--no-browser-auth` is false (the default):
  - [x] Modify `getTokenFromWeb` to start a local HTTP server on a free port.
  - [x] Update the `oauth2.Config` `RedirectURL` to point to the local server.
  - [x] Use the `github.com/pkg/browser` library to automatically open the auth URL.
  - [x] The local server will handle the redirect from Google, capture the authorization code, and send it back to the main application.
  - [x] The server should display a success message to the user in the browser and then shut down.
- [x] If `--no-browser-auth` is true, use the existing manual copy-paste flow.

### Cobra Command Structure
- [x] Refactor `main.go` to be the Cobra entry point.
- [x] Implement `cmd/root.go` to define the root command and persistent flags.

### Testing for Phase 1
- [x] Manually run the application for the first time.
- [x] Verify that the browser opens for the OAuth 2.0 consent screen.
- [x] After consent, verify that `~/.config/drivectl/token.json` is created.

## Phase 2: Core Read-Only Commands

### Command Implementation: `list`
- [x] Create `cmd/list.go`.
- [x] Implement the `drivectl list` command.
- [x] Add a `--query` (`-q`) flag to filter results based on the Google Drive API query language.
- [x] Add a `--limit` flag to control the number of results returned.

### Command Implementation: `describe`
- [x] Create `cmd/describe.go`.
- [x] Implement the `drivectl describe <file-id>` command.
- [x] The command should print detailed metadata for the given file ID.

### Testing for Phase 2
- [x] Run `drivectl list` and verify it prints a list of files from Google Drive.
- [x] Run `drivectl list -q "mimeType='application/vnd.google-apps.document'"` and verify the filtering works.
- [x] Copy a file ID from the list output.
- [x] Run `drivectl describe <file-id>` and verify it prints the correct metadata.

## Phase 3: File Content Command

### Command Implementation: `get`
- [x] Create `cmd/get.go`.
- [x] Implement the `drivectl get <file-id>` command.
- [x] Add logic to detect the file's MIME type.
- [x] For regular files (e.g., `image/jpeg`, `text/plain`), download the raw content.
- [x] For Google Docs (`application/vnd.google-apps.document`), export them as HTML from the API.
- [x] Use the `html2text` library to convert the exported HTML to clean plain text.
- [x] The output should be printed to standard output. Add a flag `--output` (`-o`) to save to a file.

### Testing for Phase 3
- [x] Find a non-Google Doc file in your Drive (e.g., a `.txt` or `.jpg`).
- [x] Run `drivectl get <file-id> -o test.jpg` and verify the file is downloaded correctly.
- [x] Find a Google Doc in your Drive.
- [x] Run `drivectl get <gdoc-id>` and verify the plain text content is printed to the console.
- [x] Run `drivectl get <gdoc-id> -o my-doc.txt` and verify the content is saved to the file.

## Phase 4: Documentation & Finalization

### README.md
- [x] Create `README.md`.
- [x] Add an overview of the tool.
- [x] Add detailed instructions on how to obtain `client_secret.json` from Google Cloud Console.
- [x] Add installation and build instructions.
- [x] Add usage examples for all commands (`list`, `describe`, `get`).

### Testing for Phase 4
- [x] Review `README.md` for clarity and accuracy.
- [x] Follow the instructions in the README from a clean state to ensure they work.

## Phase 5: Advanced Document Processing with Google Docs API

### Goal
Leverage the structured data from the Google Docs API to enable more granular access and custom exports of Google Docs content.

### Part A: Accessing Document Structure
- [x] **Dependency & Scope:**
  - [x] Add the Google Docs API Go client: `go get google.golang.org/api/docs/v1`.
  - [x] Add the `documents.readonly` scope to the authentication flow.
- [x] **`tabs` Command:**
  - [x] Create `cmd/tabs.go`.
  - [x] Implement `drivectl tabs <document-id>` to list the titles and IDs of all tabs in a document.
- [x] **Testing:**
  - [x] Run `drivectl tabs` on a multi-tab document and verify the output.

### Part B: Per-Tab Content Extraction
- [x] **Investigation:**
  - [x] Analyze the `Document` body structure to determine the mechanism for associating content elements (paragraphs, tables) with specific tabs.
- [x] **Implementation (`get --tab-index`):**
  - [x] Add a `--tab-index` flag to the `get` command.
  - [x] When used, the command will parse the document body and extract only the content related to the specified tab.

### Part C: Custom Exporters
- [ ] **Investigation:**
  - [ ] Analyze the `StructuralElement` objects (paragraphs, tables, lists, text runs, styling) to understand how to map them to different formats.
- [ ] **Implementation (`get --format custom-md`):**
  - [ ] Add new format options like `--format markdown` or `--format clean-html`.
  - [ ] Create a "renderer" that iterates through the Docs API response.
  - [ ] The renderer will convert each element into the target format (e.g., convert a `HEADING_1` paragraph to a `# Heading` in Markdown, a `BULLET` list to a `* item` list, a `TextRun` with a `bold` style to `**bold**` text, etc.).
- [ ] **Testing:**
  - [ ] Create a test document with various formatting (headings, bold, italics, lists, tables).
  - [ ] Run the custom exporter and verify the output is well-formatted Markdown/HTML/etc.
