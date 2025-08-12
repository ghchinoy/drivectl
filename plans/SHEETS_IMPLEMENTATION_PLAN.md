# Google Sheets Integration Implementation Plan

This document outlines the plan for integrating Google Sheets functionality into the `drivectl` tool.

## 1. High-Level Plan

The goal of this project is to add the ability to read data from Google Sheets. This will be implemented as a new set of Cobra commands and exposed as MCP tools.

The implementation will be divided into the following phases:

1.  **Project Setup:** Add the necessary dependencies and update the authentication scopes.
2.  **Core Logic Implementation:** Implement the functions for interacting with the Google Sheets API in the `internal/drive` package.
3.  **Cobra Command Implementation:** Create the new `sheets` and `docs` commands and their subcommands.
4.  **MCP Integration:** Expose the new commands as MCP tools.
5.  **Testing and Documentation:** Create a new test plan, update the documentation, and ensure the quality of the implementation.

## 2. Questions for Review

Before proceeding with the implementation, I would like to get your feedback on the following points:

*   **Authentication Scopes:** I'm planning to add the `sheets.SpreadsheetsReadonlyScope`. Is this scope sufficient, or do you anticipate needing to write to sheets in the future?
*   **Output Format:** For the `get-range` command, what is the desired output format? Should it be a simple CSV, or a JSON object with the values?

## 3. Detailed Plan

### Phase 1: Project Setup

- [ ] Add the `google.golang.org/api/sheets/v4` dependency to the `go.mod` file.
- [ ] Update the `internal/drive/auth.go` file to include the `sheets.SpreadsheetsReadonlyScope` in the OAuth 2.0 configuration.
- [ ] Create a new file `internal/drive/sheets.go` with skeleton functions for the Sheets API logic.

### Phase 2: Core Logic Implementation

- [ ] Implement the `ListSheets` function in `internal/drive/sheets.go` to list the sheets in a spreadsheet.
- [ ] Implement the `GetSheetAsCSV` function in `internal/drive/sheets.go` to get a sheet as CSV.
- [ ] Implement the `GetSheetRange` function in `internal/drive/sheets.go` to get a specific range from a sheet.

### Phase 3: Cobra Command Implementation

- [ ] Create a new file `cmd/sheets.go` and add the `sheets` command with its subcommands (`list`, `get`, `get-range`).
- [ ] Create a new file `cmd/docs.go` and move the `tabs` command under a new `docs` subcommand.
- [ ] Update the `root.go` file to add the new `sheets` and `docs` commands.
- [ ] Implement the `RunE` functions for these commands to call the functions in `internal/drive/sheets.go` and `internal/drive/drive.go`.

### Phase 4: MCP Integration

- [ ] Add new tool handlers in `mcp/server.go` for the new `sheets` and `docs` commands, using the `.` notation for the tool names (e.g., `sheets.list`).
- [ ] Define the `Args` structs for the new tools.
- [ ] Implement the tool handlers to call the functions in `internal/drive/sheets.go`.
- [ ] Add a new MCP resource that provides an explanation of A1 notation.

### Phase 5: Testing and Documentation

- [ ] Create a new `sheets_test_plan.md` file with manual test cases for the new functionality.
- [ ] Update the `README.md` file to document the new `sheets` and `docs` commands.
- [ ] Update the `MCP_IMPLEMENTATION_PLAN.md` file with the new plan.
- [ ] Create a `.commit.txt` file with a summary of the changes.