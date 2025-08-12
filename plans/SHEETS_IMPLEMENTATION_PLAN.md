# Google Sheets Integration Implementation Plan

This document outlines the plan for integrating Google Sheets functionality into the `drivectl` tool.

## 1. High-Level Plan

The goal of this project is to add the ability to read and write data from Google Sheets. This will be implemented as a new set of Cobra commands and exposed as MCP tools.

The implementation will be divided into the following phases:

1.  **Project Setup:** Add the necessary dependencies and update the authentication scopes.
2.  **Core Logic Implementation:** Implement the functions for interacting with the Google Sheets API in the `internal/drive` package.
3.  **Cobra Command Implementation:** Create the new `sheets` and `docs` commands and their subcommands.
4.  **MCP Integration:** Expose the new commands as MCP tools.
5.  **Testing and Documentation:** Create a new test plan, update the documentation, and ensure the quality of the implementation.

## 2. Questions for Review

Before proceeding with the implementation, I would like to get your feedback on the following points:

*   **Authentication Scopes:** I'm planning to add the `sheets.SpreadsheetsScope`. Is this scope sufficient for both read and write operations?
*   **Output Format:** For the `get-range` command, the current output is a string representation of a 2D slice. Is this sufficient, or should it be formatted as CSV or JSON?

## 3. Detailed Plan

### Phase 1: Project Setup

- [x] Add the `google.golang.org/api/sheets/v4` dependency to the `go.mod` file.
- [x] Update the `internal/drive/auth.go` file to include the `sheets.SpreadsheetsReadonlyScope` in the OAuth 2.0 configuration.
- [x] Create a new file `internal/drive/sheets.go` with skeleton functions for the Sheets API logic.

### Phase 2: Core Logic Implementation

- [x] Implement the `ListSheets` function in `internal/drive/sheets.go` to list the sheets in a spreadsheet.
- [x] Implement the `GetSheetAsCSV` function in `internal/drive/sheets.go` to get a sheet as CSV.
- [x] Implement the `GetSheetRange` function in `internal/drive/sheets.go` to get a specific range from a sheet.
- [x] Implement the `UpdateSheetRange` function in `internal/drive/sheets.go` to update a specific range in a sheet.

### Phase 3: Cobra Command Implementation

- [x] Create a new file `cmd/sheets.go` and add the `sheets` command with its subcommands (`list`, `get`, `get-range`, `update-range`).
- [x] Create a new file `cmd/docs.go` and move the `tabs` command under a new `docs` subcommand.
- [x] Update the `root.go` file to add the new `sheets` and `docs` commands.
- [x] Implement the `RunE` functions for these commands to call the functions in `internal/drive/sheets.go` and `internal/drive/drive.go`.
- [x] Add the `update-range` subcommand to `cmd/sheets.go`.

### Phase 4: MCP Integration

- [x] Add new tool handlers in `mcp/server.go` for the new `sheets` and `docs` commands, using the `.` notation for the tool names (e.g., `sheets.list`).
- [x] Define the `Args` structs for the new tools.
- [x] Implement the tool handlers to call the functions in `internal/drive/sheets.go`.
- [x] Add a new MCP resource that provides an explanation of A1 notation.

### Phase 5: Testing and Documentation

- [x] Create a new `sheets_test_plan.md` file with manual test cases for the new functionality.
- [x] Update the `sheets_test_plan.md` with the current status.
- [x] Update the `README.md` file to document the new `sheets` and `docs` commands.
- [x] Update the `MCP_IMPLEMENTATION_PLAN.md` file with the new plan.
- [x] Create a `.commit.txt` file with a summary of the changes.