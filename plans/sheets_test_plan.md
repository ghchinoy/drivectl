# Manual Test Plan for Google Sheets Integration

This document outlines the manual test cases to verify the Google Sheets functionality of the `drivectl` tool.

### Prerequisites

- [x] A Google Sheet with at least two sheets and some data.
- [x] The spreadsheet ID of the test sheet.
- [x] A Google Doc.
- [x] The document ID of the test doc.

---

### Test Case 1: `sheets list` command

- [x] **Action:** Run `drivectl sheets list <spreadsheet-id>`
- [x] **Verification:** Does the command print a list of all sheets in the spreadsheet?

---

### Test Case 2: `sheets get` command

- [x] **Action:** Run `drivectl sheets get <spreadsheet-id> --sheet <sheet-name>`
- [x] **Verification:** Does the command print the content of the specified sheet as CSV?

---

### Test Case 3: `sheets get-range` command

- [x] **Action:** Run `drivectl sheets get-range <spreadsheet-id> --sheet <sheet-name> --range <A1-notation>`
- [x] **Verification:** Does the command print the content of the specified range?

---

### Test Case 4: `sheets update-range` command

- [x] **Action:** Run `drivectl sheets update-range <spreadsheet-id> --sheet <sheet-name> --range <A1-notation> --values <values>`
- [x] **Verification:** Does the command update the specified range?
- [x] **Verification:** Run `get-range` again to confirm the update.

---

### Test Case 5: `docs tabs` command

- [x] **Action:** Run `drivectl docs tabs <document-id>`
- [x] **Verification:** Does the command print the list of tabs in the document?

---

### Test Case 6: MCP Tools

- [x] **Action:** Start the MCP server: `drivectl --mcp`
- [x] **Action:** List the available tools: `mcptools tools ./drivectl --mcp`
- [x] **Verification:** Are the new `sheets.list`, `sheets.get`, `sheets.get-range`, and `docs.tabs` tools listed?
- [x] **Action:** Call the `sheets.list` tool.
- [x] **Verification:** Does the tool return the list of sheets?
- [x] **Action:** Call the `sheets.get` tool.
- [x] **Verification:** Does the tool return the content of the sheet as CSV?
- [x] **Action:** Call the `sheets.get-range` tool.
- [x] **Verification:** Does the tool return the content of the range?
- [x] **Action:** Call the `docs.tabs` tool.
- [x] **Verification:** Does the tool return the list of tabs?

---

### Test Case 7: MCP Resource

- [x] **Action:** List the available resources: `mcptools resources ./drivectl --mcp`
- [x] **Verification:** Is the new resource for A1 notation listed?
- [x] **Action:** Read the A1 notation resource.
- [x] **Verification:** Is the content of the resource correct?