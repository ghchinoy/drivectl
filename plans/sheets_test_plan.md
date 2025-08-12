# Manual Test Plan for Google Sheets Integration

This document outlines the manual test cases to verify the Google Sheets functionality of the `drivectl` tool.

### Prerequisites

- [ ] A Google Sheet with at least two sheets and some data.
- [ ] The spreadsheet ID of the test sheet.
- [ ] A Google Doc.
- [ ] The document ID of the test doc.

---

### Test Case 1: `sheets list` command

- [ ] **Action:** Run `drivectl sheets list`
- [ ] **Verification:** Does the command print a list of all Google Sheets in your Drive?

---

### Test Case 2: `sheets get` command

- [ ] **Action:** Run `drivectl sheets get <spreadsheet-id> --sheet <sheet-name>`
- [ ] **Verification:** Does the command print the content of the specified sheet as CSV?

---

### Test Case 3: `sheets get-range` command

- [x] **Action:** Run `drivectl sheets get-range <spreadsheet-id> --sheet <sheet-name> --range <A1-notation>`
- [x] **Verification:** Does the command print the content of the specified range?

---

### Test Case 4: `sheets update-range` command (To Be Implemented)

- [ ] **Action:** Run `drivectl sheets update-range <spreadsheet-id> --sheet <sheet-name> --range <A1-notation> --values <values>`
- [ ] **Verification:** Does the command update the specified range?
- [ ] **Verification:** Run `get-range` again to confirm the update.

---

### Test Case 5: `docs list` command

- [ ] **Action:** Run `drivectl docs list`
- [ ] **Verification:** Does the command print a list of all Google Docs in your Drive?

---

### Test Case 6: `docs tabs` command

- [ ] **Action:** Run `drivectl docs tabs <document-id>`
- [ ] **Verification:** Does the command print the list of tabs in the document?

---

### Test Case 7: MCP Tools

- [ ] **Action:** Start the MCP server: `drivectl --mcp`
- [ ] **Action:** List the available tools: `mcptools tools ./drivectl --mcp`
- [ ] **Verification:** Are the new `sheets.list`, `sheets.get`, `sheets.get-range`, `docs.list`, and `docs.tabs` tools listed?
- [ ] **Action:** Call the `sheets.list` tool.
- [ ] **Verification:** Does the tool return the list of sheets?
- [ ] **Action:** Call the `sheets.get` tool.
- [ ] **Verification:** Does the tool return the content of the sheet as CSV?
- [ ] **Action:** Call the `sheets.get-range` tool.
- [ ] **Verification:** Does the tool return the content of the range?
- [ ] **Action:** Call the `docs.list` tool.
- [ ] **Verification:** Does the tool return the list of docs?
- [ ] **Action:** Call the `docs.tabs` tool.
- [ ] **Verification:** Does the tool return the list of tabs?

---

### Test Case 8: MCP Resource

- [ ] **Action:** List the available resources: `mcptools resources ./drivectl --mcp`
- [ ] **Verification:** Is the new resource for A1 notation listed?
- [ ] **Action:** Read the A1 notation resource.
- [ ] **Verification:** Is the content of the resource correct?