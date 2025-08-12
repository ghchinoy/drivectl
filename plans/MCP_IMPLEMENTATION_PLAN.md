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

### Phase 3: Cobra Command Implementation

- [x] Create a new file `cmd/sheets.go` and add the `sheets` command with its subcommands (`list`, `get`, `get-range`, `update-range`).
- [x] Create a new file `cmd/docs.go` and move the `tabs` command under a new `docs` subcommand.
- [x] Update the `root.go` file to add the new `sheets` and `docs` commands.
- [x] Implement the `RunE` functions for these commands to call the functions in `internal/drive/sheets.go` and `internal/drive/drive.go`.

### Phase 4: MCP Integration

- [x] Add new tool handlers in `mcp/server.go` for the new `sheets` and `docs` commands, using the `.` notation for the tool names (e.g., `sheets.list`).
- [x] Define the `Args` structs for the new tools.
- [x] Implement the tool handlers to call the functions in `internal/drive/sheets.go`.
- [x] Add a new MCP resource that provides an explanation of A1 notation.

### Phase 5: Documentation and Testing

- [x] Create a new `sheets_test_plan.md` file with manual test cases for the new functionality.
- [x] Update the `README.md` file to document the new `sheets` and `docs` commands.
- [x] Update this `MCP_IMPLEMENTATION_PLAN.md` file with the current status.
- [x] Create a `.commit.txt` file with a summary of the changes.

## 3. Guide: Creating MCP Servers from Cobra CLIs

Integrating a Cobra-based command-line tool with the MCP Go SDK is a powerful way to expose your CLI's functionality to other programs. This guide provides a set of best practices and lessons learned from the implementation of the `drivectl` MCP server.

### 1. Core Principle: Decouple Logic from UI

The most important principle is to decouple your core application logic from your command-line interface (CLI) code.

**Don't:** Put your API call logic directly inside your Cobra `RunE` functions.

**Do:** Create a separate package (e.g., `internal/applogic`) that contains all the core functionality. Your Cobra commands and your MCP tool handlers should both call into this package.

**Benefits:**
*   **Reusability:** The same core logic can be used by the CLI and the MCP server.
*   **Testability:** The core logic can be tested independently of the CLI.
*   **Maintainability:** The code is cleaner, more modular, and easier to understand.

### 2. Structuring Your MCP Server

The `mcp/server.go` file is the heart of your MCP server. Here's a good way to structure it:

*   **Service Initializers:** Create separate functions (e.g., `getDriveSvc`, `getDocsSvc`) to initialize your API services. These functions should handle authentication and client creation.
*   **Argument Structs:** For each MCP tool, define a struct that represents its arguments (e.g., `ListArgs`, `GetArgs`). Use JSON tags to map the struct fields to the JSON parameters that the MCP client will send.
*   **Tool Handlers:** The tool handlers are the functions that are executed when an MCP client calls a tool. They should be responsible for:
    1.  Parsing the arguments from the `params` object.
    2.  Calling the appropriate function in your core logic package.
    3.  Formatting the result and returning it in a `mcp.CallToolResultFor` struct.

### 3. Mapping Cobra Commands to MCP Tools

A common pattern is to iterate through your Cobra commands and create a corresponding MCP tool for each one.

*   **Subcommands:** For commands with subcommands (like `drivectl sheets list`), use a `.` separator in the MCP tool name to create a 