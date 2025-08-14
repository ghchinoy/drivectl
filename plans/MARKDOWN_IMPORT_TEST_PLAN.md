# Markdown Import Test Plan

This document outlines the manual test cases for verifying the Markdown import functionality.

## Prerequisites

- [ ] A sample Markdown file (`test.md`) with a variety of elements:
    - Headings (H1, H2, H3)
    - Paragraphs with **bold** and *italic* text
    - A bulleted list
    - A numbered list
    - A hyperlink
- [ ] The `drivectl` binary built from the `feat/docs-markdown-import` branch.

---

## Phase 1: The Converter (Unit Tests)

- [ ] **Action:** Create a "reference" Google Doc with all the target formatting.
- [ ] **Action:** Use `drivectl get <reference-doc-id> --format json` to get the JSON representation of the reference document.
- [ ] **Action:** Add unit tests for the Markdown-to-Docs-JSON converter in `internal/drive`.
- [ ] **Verification:** Do the unit tests pass for all supported Markdown elements, comparing the output to the JSON from the reference document?

---

## Phase 2: `docs create` Command

- [x] **Action:** Run `drivectl docs create --title "Markdown Test" --from-markdown test.md`
- [x] **Verification:**
    - [x] Is a new Google Doc named "Markdown Test" created in your Google Drive?
    - [x] Does the content of the new Google Doc correctly reflect the formatting from the `test.md` file?
        - [x] Are headings rendered correctly?
        - [x] Are bold and italic styles applied correctly?
        - [x] Are lists formatted correctly?
        - [x] Is the hyperlink clickable and does it point to the correct URL?

---

## Phase 3: `docs.create` MCP Tool

### Known Issues
*   When passing Markdown text directly to the `docs.create` tool using the `markdown_text` parameter, newline characters (`\n`) are not always interpreted correctly by the shell, which can result in the text being rendered as a single line.

- [ ] **Action:** Run `mcptools call docs.create -p '{"title": "MCP Markdown Test", "markdown_file": "test.md"}' ./drivectl --mcp`
- [ ] **Verification:**
    - [ ] Is a new Google Doc named "MCP Markdown Test" created?
    - [ ] Does the content of the new doc match the formatting from the `test.md` file?

---

## Phase 4: Add/Replace Tab (Advanced) - Not Feasible

As noted in the implementation plan, the Google Docs API does not currently support the programmatic creation of new tabs. This section of the test plan is therefore not applicable.

---

## Phase 5: Regression Testing

- [ ] **Action:** Run all tests from `sheets_test_plan.md`.
- [ ] **Verification:** Do all the `sheets` and existing `docs` commands still work as expected?