# Dynamic MCP Tools Verification Test Plan

This document outlines the steps to verify the newly refactored Model Context Protocol (MCP) server, which now dynamically generates its tools and schemas from Google API Discovery Documents.

## Prerequisites
* You must have already authenticated using `./drivectl auth login`.
* You need an MCP client like `mcptools` or the Gemini CLI installed to test the stdio transport. Alternatively, `curl` can be used to test the HTTP transport.

## 1. Verify Tool Generation & Initialization

When the MCP server starts, it should fetch Discovery Documents and dynamically generate schemas for tools like `drive.v3.files.list` instead of the old hardcoded `list`.

### Using `mcptools` (over stdio)
1. **List available tools:**
   ```bash
   mcptools tools ./drivectl --mcp
   ```
   *Expected Result:* You should see a list of dynamically generated tools matching the Google API endpoints, such as:
   - `drive.v3.files.list`
   - `drive.v3.files.get`
   - `docs.v1.documents.create`
   - `sheets.v4.spreadsheets.values.update`

### Using HTTP Server Mode
1. **Start the server in HTTP mode:**
   ```bash
   ./drivectl --mcp-http :8080 &
   ```
2. **Fetch the tools list via HTTP:**
   ```bash
   curl -s http://localhost:8080/tools
   ```
   *Expected Result:* The JSON response should contain a `tools` array populated with the dynamically named tools and their intricate JSON schemas.
3. **Kill the background server:**
   ```bash
   kill %1
   ```

## 2. Test a Read Operation (`drive.v3.files.list`)

Let's test an API endpoint that only requires query parameters.

1. **Call the tool:**
   ```bash
   mcptools call drive.v3.files.list -p '{"pageSize": 2}' ./drivectl --mcp
   ```
   *Expected Result:* The MCP tool should execute successfully, returning a JSON response with up to 2 files from your Google Drive. Note that the output should be valid JSON containing a `files` array.

## 3. Test Path Parameters (`drive.v3.files.get`)

Let's test if the MCP server correctly extracts parameters from the payload and injects them into the URL path.

1. **Get a File ID:**
   Grab an ID from the list you just generated.
2. **Call the tool:**
   ```bash
   mcptools call drive.v3.files.get -p '{"fileId": "YOUR_FILE_ID", "fields": "name,mimeType"}' ./drivectl --mcp
   ```
   *Expected Result:* The tool succeeds and returns the file metadata. The server internally routed `fileId` to the path and `fields` to the query string.

## 4. Test a Write Operation (`docs.v1.documents.create`)

Let's test a tool that expects a complex nested JSON body payload. Based on the discovery doc, `docs.v1.documents.create` expects a `Document` schema which we mapped under the `payload` key to separate it from path parameters.

1. **Call the tool:**
   ```bash
   mcptools call docs.v1.documents.create -p '{"payload": {"title": "MCP Dynamic Creation Test"}}' ./drivectl --mcp
   ```
   *Expected Result:* The command succeeds and returns the JSON representation of the newly created Google Doc.

## 5. Test Error Handling (Validation)

1. **Omit a Required Path Parameter:**
   ```bash
   mcptools call drive.v3.files.get -p '{"fields": "name"}' ./drivectl --mcp
   ```
   *Expected Result:* The tool should immediately return an error from the Go code stating `missing required path parameter: fileId`, before attempting the network call.