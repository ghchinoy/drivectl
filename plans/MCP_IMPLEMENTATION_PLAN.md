# MCP Integration Implementation Plan

This document outlines the design, phases, and tasks for integrating an MCP server into the `drivectl` tool.

## 1. Design

The goal of this project is to expose the existing `drivectl` CLI commands as tools in an MCP server, without affecting the current CLI functionality. This will be achieved by adding a new mode to the application, triggered by the `--mcp` or `--mcp-http` flags.

### 1.1. Architecture

The new architecture will be as follows:

*   **`main.go`:** The entry point of the application. It will be modified to check for the `--mcp` and `--mcp-http` flags. If either is present, it will call `cmd.ExecuteMCP()`. Otherwise, it will call `cmd.Execute()` as it does now.
*   **`cmd/root.go`:** The root Cobra command. It will be modified to add the `--mcp` and `--mcp-http` flags.
*   **`cmd/mcp.go`:** A new file that will contain the `ExecuteMCP()` function. This function will be responsible for initializing the MCP server by calling `mcp.Start()`.
*   **`internal/drive/drive.go`:** A new package that abstracts the logic for interacting with the Google Drive and Docs APIs.
*   **`mcp/server.go`:** A new package and file that will contain the core MCP server logic. The `Start()` function in this file will:
    *   Create a new `mcp.Server`.
    *   Iterate through the Cobra commands of `drivectl`.
    *   For each command, create an `mcp.Tool` with the command's name, description, and a handler function.
    *   The handler function will call the corresponding function in the `internal/drive` package.
    *   Start the MCP server on stdio or HTTP, based on the provided flags.

### 1.2. Data Flow

1.  The user starts `drivectl` with `--mcp` or `--mcp-http`.
2.  `main.go` detects the flag and calls `cmd.ExecuteMCP()`.
3.  `cmd.ExecuteMCP()` calls `mcp.Start()`.
4.  `mcp.Start()` creates and configures the MCP server.
5.  The MCP server starts listening for requests (on stdio or HTTP).
6.  An MCP client (e.g., `mcptools`) sends a `call-tool` request.
7.  The MCP server receives the request and calls the corresponding tool handler.
8.  The tool handler calls the corresponding function in the `internal/drive` package, captures the output, and returns it as the tool result.
9.  The MCP server sends the tool result back to the client.

## 2. Phases and Tasks

### Phase 1: Project Setup and Scaffolding

- [x] Create the `mcp` directory.
- [x] Create the `mcp/server.go` file with a skeleton `Start` function.
- [x] Create the `cmd/mcp.go` file with a skeleton `ExecuteMCP` function.
- [x] Add the `--mcp` and `--mcp-http` flags to `cmd/root.go`.
- [x] Modify `main.go` to call `cmd.ExecuteMCP()` when the flags are present.

### Phase 2: MCP Server Implementation

- [x] In `mcp/server.go`, implement the `Start` function:
    - [x] Create a new `mcp.Server`.
    - [x] Iterate through the Cobra commands and create `mcp.Tool`s for each.
    - [x] Implement the tool handler function to execute the Cobra commands and capture their output.
- [x] Implement the logic to start the server on stdio (`--mcp`).
- [x] Implement the logic to start the server on HTTP (`--mcp-http`).
- [x] Refactor the code to abstract the Drive and Docs APIs into an `internal/` package.

### Phase 3: Testing and Verification

- [x] Execute the manual test plan in `plans/mcp_test_plan.md`.
- [x] **Regression Testing:**
    - [x] Verify that the existing CLI functionality is unaffected.
- [x] **MCP Stdio Mode Testing:**
    - [x] Test listing tools.
    - [x] Test calling tools with and without arguments.
- [x] **MCP HTTP Mode Testing:**
    - [x] Test listing tools.
    - [x] Test calling tools with and without arguments.
- [x] **Error Handling Testing:**
    - [x] Test calling tools with invalid arguments.
    - [x] Test calling non-existent tools.

### Phase 4: Documentation and Cleanup

- [ ] Update the `README.md` to document the new MCP server functionality.
- [ ] Review and refactor the code for clarity and maintainability.
- [x] Add Go doc comments to all the methods.
- [x] Make sure the long command descriptions are being used in `mcp/server.go`.
- [ ] Add a new MCP resource with a cheat sheet of Drive query examples.

## 3. Lessons Learned

### Phase 1:

*   No major lessons learned in this phase. The scaffolding was straightforward.

### Phase 2:

*   **Cobra and MCP Integration:** Executing subcommands in MCP mode is not as straightforward as calling `command.Execute()`. The `PersistentPreRunE` of the root command is not automatically called. The initial approach of calling `rootCmd.Execute()` with the subcommand and its arguments did not work as expected. The better approach is to abstract the core logic into a separate package that can be called from both the Cobra commands and the MCP tool handlers. This makes the code more modular, easier to test, and avoids the complexities of the Cobra command execution flow.
*   **Viper and Environment Variables:** When running in MCP mode, where the `drivectl` process is a child process of `mcptools`, it's important to ensure that `viper` is initialized correctly to read environment variables. Calling `viper.AutomaticEnv()` at the beginning of the `toolHandler` is a good practice.
*   **Debugging MCP Servers:** Debugging MCP servers can be tricky, as the server runs as a child process and its stdout/stderr are not always visible. Redirecting logs to a file is a useful technique for debugging.
*   **Google Drive API Nuances:** The Google Drive API has different methods for downloading binary files and exporting Google Docs. It's important to check the mime type of the file and use the appropriate method. The supported export formats are different for different Google Docs types. For example, Google Sheets cannot be exported as `text/plain`.
*   **MCP Go SDK:** The `mcp-go` SDK is still under development and the documentation is not always complete. It's important to read the source code of the SDK to understand how to use it correctly. The generic `AddTool` function is the recommended way to add tools to the server. The `CallToolResultFor` struct has a `StructuredContent` field for returning structured data, and a `Content` field for returning unstructured data. The `mcptools` CLI is a useful tool for testing MCP servers, but it has some limitations (e.g., it doesn't have a flag to specify the server address for HTTP transport, and it expects the `content` field to be an array).

### Phase 3:

*   The `mcptools` CLI is not suitable for testing HTTP-based MCP servers. `curl` can be used as an alternative, but it requires manual construction of JSON-RPC requests and handling of session IDs.

### Phase 4:

*   ...
