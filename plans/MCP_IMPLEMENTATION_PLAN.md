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
- [x] Add a new MCP resource with a cheat sheet of Drive query examples.

## 3. Lessons Learned

### Cobra and MCP Integration: A Deep Dive

Integrating a Cobra-based command-line tool with the MCP Go SDK presents a unique set of challenges and learning opportunities. Here's a more detailed breakdown of the lessons learned during this process:

**1. The Challenge of Command Execution in MCP Mode**

The most significant hurdle was correctly executing the Cobra subcommands from within the MCP tool handlers. A naive approach of calling `command.Execute()` on the subcommand itself does not work as expected. This is because the `PersistentPreRunE` function of the root command, which is responsible for initializing the Google Drive and Docs services, is not automatically called when a subcommand is executed directly.

The initial attempt to solve this by manually calling `rootCmd.PersistentPreRunE(command, args)` also failed. This is because the context of the root command is not properly propagated to the subcommand when it's executed in this manner.

**The Solution: Abstracting Core Logic**

The most effective solution was to refactor the code and abstract the core logic for interacting with the Google Drive and Docs APIs into a separate `internal/drive` package. This approach has several key advantages:

*   **Decoupling:** It decouples the command-line interface (Cobra) from the core application logic. This makes the code cleaner, more modular, and easier to maintain.
*   **Reusability:** The functions in the `internal/drive` package can be easily reused by both the Cobra commands and the MCP tool handlers.
*   **Testability:** The core logic can be tested independently of the command-line interface.
*   **Simplified Tool Handlers:** The MCP tool handlers become much simpler. They are only responsible for parsing the tool arguments, calling the corresponding function in the `internal/drive` package, and formatting the result.

Here's an example of how the `list` command was refactored:

**Before:**

```go
// cmd/list.go
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists files and folders in Google Drive.",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := driveSvc.Files.List().PageSize(limit).Q(query).
			Fields("nextPageToken, files(id, name)").Do()
		if err != nil {
			return fmt.Errorf("unable to retrieve files: %w", err)
		}
		// ... print files
		return nil
	},
}
```

**After:**

```go
// internal/drive/drive.go
func ListFiles(srv *drive.Service, limit int64, query string) ([]*drive.File, error) {
	r, err := srv.Files.List().PageSize(limit).Q(query).
		Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		return nil, err
	}
	return r.Files, nil
}

// cmd/list.go
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists files and folders in Google Drive.",
	RunE: func(cmd *cobra.Command, args []string) error {
		files, err := drive.ListFiles(driveSvc, limit, query)
		if err != nil {
			return fmt.Errorf("unable to retrieve files: %w", err)
		}
		// ... print files
		return nil
	},
}

// mcp/server.go
mcp.AddTool(server, &mcp.Tool{
	Name:        "list",
	Description: "Lists files and folders in Google Drive.",
}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListArgs]) (*mcp.CallToolResultFor[any], error) {
	driveSvc, err := getDriveSvc(ctx)
	if err != nil {
		return nil, err
	}
	files, err := drive.ListFiles(driveSvc, params.Arguments.Limit, params.Arguments.Query)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve files: %w", err)
	}
	// ... format and return result
})
```

**2. Viper and Environment Variables in MCP Mode**

When running `drivectl` in MCP mode, the `drivectl` process is a child process of `mcptools`. This means that it doesn't automatically inherit the environment variables of the parent shell.

To solve this, it's important to ensure that `viper` is initialized correctly to read environment variables. Calling `viper.AutomaticEnv()` at the beginning of the `getDriveSvc` and `getDocsSvc` functions ensures that the `DRIVE_SECRETS` environment variable is read correctly.

**3. Debugging MCP Servers**

Debugging MCP servers can be challenging due to the client-server architecture and the fact that the server often runs as a child process. Here are some useful debugging techniques:

*   **Logging to a file:** Redirecting the server's logs to a file is an effective way to capture the output and debug issues.
*   **Using `curl`:** For HTTP-based MCP servers, `curl` can be used to send raw JSON-RPC requests and inspect the responses. This is particularly useful when the `mcptools` CLI does not support a specific feature (e.g., HTTP transport).
*   **Reading the SDK source code:** The `mcp-go` SDK is still under development, and the documentation is not always complete. Reading the source code is often the best way to understand how to use it correctly.

**4. MCP Go SDK Nuances**

The `mcp-go` SDK has some nuances that are important to be aware of:

*   **Generic `AddTool`:** The generic `AddTool` function is the recommended way to add tools to the server. It automatically infers the input and output schemas from the tool handler's signature, which simplifies the process of defining tools.
*   **`CallToolResultFor` struct:** The `CallToolResultFor` struct has a `StructuredContent` field for returning structured data and a `Content` field for returning unstructured data. It's important to use the correct field based on the type of data you are returning.
*   **`mcptools` limitations:** The `mcptools` CLI is a useful tool for testing MCP servers, but it has some limitations. For example, it doesn't have a flag to specify the server address for the HTTP transport, and it expects the `content` field in the tool result to be an array.

By following these lessons learned, developers can more easily integrate their Cobra-based command-line tools with the MCP Go SDK and build powerful, flexible MCP servers.
