# Manual Test Plan for drivectl MCP Mode

This document outlines the manual test cases to verify the MCP server functionality of the `drivectl` tool. Check off items as you complete them.

### Prerequisites

- [x] Build the latest version of the tool: `go build -o drivectl .`
- [x] Have a valid `secrets.json` file and `token.json` file for authentication.
- [x] Install the `mcptools` CLI for interacting with the MCP server.

---

### Test Case 1: Regression Testing (Existing CLI)

- [x] **Action:** Run each of the following commands without any MCP flags:
  - [x] `./drivectl list`
  - [x] `./drivectl describe <file-id>`
  - [x] `./drivectl get <file-id>`
  - [x] `./drivectl tabs <file-id>` (**Note:** This command failed, but it's a pre-existing bug in the command itself, not a regression.)
- [x] **Verification:**
    - [x] Does each command execute successfully and produce the expected output?
    - [x] Is there any change in behavior from the previous version? (There should not be.)

---

### Test Case 2: MCP Stdio Mode

- [x] **Action:** Start the server in stdio mode: `./drivectl --mcp`
- [x] **Action:** In a separate terminal, run `mcptools tools ./drivectl --mcp`
- [x] **Verification:**
    - [x] Are all the `drivectl` commands (`list`, `describe`, `get`, `tabs`) listed as available tools?
- [x] **Action:** Call the `list` tool: `mcptools call list ./drivectl --mcp`
- [x] **Verification:** Is the output a valid JSON response containing a list of your Drive files? (**Status:** Passing)
- [x] **Action:** Call the `get` tool with a file ID: `mcptools call get -p '{"file-id": "<your-file-id>"}' ./drivectl --mcp`
- [x] **Verification:** Does the tool return the content of the file? (**Status:** Passing)
- [x] **Action:** Call the `describe` tool with a file ID: `mcptools call describe -p '{"file-id": "<your-file-id>"}' ./drivectl --mcp`
- [x] **Verification:** Does the tool return the file metadata? (**Status:** Passing)
- [x] **Action:** Call the `tabs` tool with a document ID: `mcptools call tabs -p '{"document-id": "<your-document-id>"}' ./drivectl --mcp`
- [x] **Verification:** Does the tool return the list of tabs? (**Status:** Passing with expected error for non-doc files)

---

### Test Case 3: MCP HTTP Mode

- [x] **Action:** Start the server in HTTP mode: `./drivectl --mcp-http :8080`
- [x] **Action:** In a separate terminal, run `curl` to list tools.
- [x] **Verification:**
    - [x] Are all the `drivectl` commands listed as available tools?
- [x] **Action:** Call the `list` tool with `curl`.
- [x] **Verification:** Is the output a valid JSON response containing a list of your Drive files?
- [x] **Action:** Call the `get` tool with a file ID with `curl`.
- [x] **Verification:** Does the tool return the content of the file?
- [x] **Action:** Call the `describe` tool with a file ID with `curl`.
- [x] **Verification:** Does the tool return the file metadata?
- [ ] **Action:** Call the `tabs` tool with a document ID with `curl`.
- [ ] **Verification:** Does the tool return the list of tabs?

---

### Test Case 4: Error Handling

- [x] **Action:** Call a tool with a missing required argument (e.g., `mcptools call get ./drivectl --mcp`)
- [x] **Verification:** Does the server return a clear error message indicating the missing argument?
- [x] **Action:** Call a tool that does not exist (e.g., `mcptools call non-existent-tool ./drivectl --mcp`)
- [x] **Verification:** Does the server return a "tool not found" error?

---

### Test Case 5: MCP Resource Testing

- [ ] **Action:** Add a new resource to the MCP server that provides a cheat sheet of Drive query examples.
- [ ] **Action:** Use `mcptools resources ./drivectl --mcp` to list the available resources.
- [ ] **Verification:** Is the new resource listed?
- [ ] **Action:** Use `mcptools read-resource <resource-name> ./drivectl --mcp` to read the resource.
- [ ] **Verification:** Is the content of the resource correct?
