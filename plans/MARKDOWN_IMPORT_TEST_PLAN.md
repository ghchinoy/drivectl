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

- [ ] **Action:** Add unit tests for the Markdown-to-Docs-JSON converter in `internal/drive`.
- [ ] **Verification:** Do the unit tests pass for all supported Markdown elements?

---

## Phase 2: `docs create` Command

- [ ] **Action:** Run `drivectl docs create --title "Markdown Test" --from-markdown test.md`
- [ ] **Verification:**
    - [ ] Is a new Google Doc named "Markdown Test" created in your Google Drive?
    - [ ] Does the content of the new Google Doc correctly reflect the formatting from the `test.md` file?
        - [ ] Are headings rendered correctly?
        - [ ] Are bold and italic styles applied correctly?
        - [ ] Are lists formatted correctly?
        - [ ] Is the hyperlink clickable and does it point to the correct URL?

---

## Phase 3: `docs.create` MCP Tool

- [ ] **Action:** Run `mcptools call docs.create -p '{"title": "MCP Markdown Test", "markdown_file": "test.md"}' ./drivectl --mcp`
- [ ] **Verification:**
    - [ ] Is a new Google Doc named "MCP Markdown Test" created?
    - [ ] Does the content of the new doc match the formatting from the `test.md` file?

---

## Phase 4: Add/Replace Tab (Advanced)

*This section will be filled out if and when the advanced "add/replace tab" functionality is implemented.*

---

## Phase 5: Regression Testing

- [ ] **Action:** Run all tests from `sheets_test_plan.md`.
- [ ] **Verification:** Do all the `sheets` and existing `docs` commands still work as expected?