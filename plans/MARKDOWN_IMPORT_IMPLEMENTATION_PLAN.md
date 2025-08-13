# Markdown Import Implementation Plan

This document outlines the plan for adding the capability to import Markdown content into Google Docs to the `drivectl` tool.

## 1. Goal

The primary goal is to allow users to create new Google Docs or update existing ones using local Markdown files as the content source. This will be exposed through both new Cobra CLI commands and new MCP tools.

This feature will be implemented by creating a "Markdown to Google Docs JSON" converter, as the Google Docs API does not support direct Markdown import.

## 2. Core Logic: The Markdown-to-Docs-JSON Converter

This is the most critical and complex part of the implementation. It will be a new set of functions in the `internal/drive` package.

*   **Markdown Parsing:** A robust Markdown parsing library will be used (e.g., `goldmark`) to parse the Markdown content into an Abstract Syntax Tree (AST).
*   **AST to Docs JSON Translation:** A new "translator" will be written to traverse the AST and convert each node type into the corresponding Google Docs `Document` object structure. This will need to handle:
    *   Headings (H1, H2, etc.)
    *   Paragraphs
    *   Bold and Italic text
    *   Bulleted and numbered lists
    *   Links
    *   Images (this will require uploading the image to Google Drive and then embedding it)
    *   Tables (if supported by the parser)

## 3. Scenarios

### Scenario 1: Create a New Google Doc from a Markdown File

*   **CLI Command:** `drivectl docs create --title "My New Doc" --from-markdown /path/to/file.md`
*   **MCP Tool:** `docs.create`
    *   **Arguments:** `title` (string), `markdown_file` (string)
*   **Implementation:**
    1.  Read the content of the local Markdown file.
    2.  Use the converter to translate the Markdown to a `*docs.Document` object.
    3.  Call the `docs.documents.create` method of the Google Docs API with the `*docs.Document` object.

### Scenario 2: Add a New Tab in an Existing Doc from a Markdown File

*   **CLI Command:** `drivectl docs add-tab <document-id> --title "My New Tab" --from-markdown /path/to/file.md`
*   **MCP Tool:** `docs.add-tab`
    *   **Arguments:** `document-id` (string), `title` (string), `markdown_file` (string)
*   **Implementation:**
    1.  This is a complex operation. The Google Docs API does not have a simple "add tab" method. A "tab" in a Google Doc is just a section of the document with a special bookmark.
    2.  We would need to:
        a.  Get the existing document.
        b.  Find the end of the document.
        c.  Insert a page break.
        d.  Insert the new content (translated from Markdown).
        e.  Create a bookmark for the new "tab".
    3.  This will require using the `documents.batchUpdate` method with a series of requests.

### Scenario 3: Replace a Tab in an Existing Doc with Content from a Markdown File

*   **CLI Command:** `drivectl docs replace-tab <document-id> --tab-index <index> --from-markdown /path/to/file.md`
*   **MCP Tool:** `docs.replace-tab`
    *   **Arguments:** `document-id` (string), `tab-index` (int), `markdown_file` (string)
*   **Implementation:**
    1.  Similar to "add tab", this is a complex operation that requires `documents.batchUpdate`.
    2.  We would need to:
        a.  Get the document and find the start and end indexes of the specified tab.
        b.  Delete the content of that tab.
        c.  Insert the new content (translated from Markdown) at the start index of the tab.

## 4. Implementation Phases

1.  **Phase 1: The Converter**
    *   **Research and Decision:**
        *   **Google Drive API `files.create`:** We will first test if the Google Drive API can convert Markdown to a Google Doc by setting the source MIME type to `text/markdown` and the destination MIME type to `application/vnd.google-apps.document`. This would be the simplest solution if it works.
        *   **Markdown Parsing Library:** If the direct conversion is not possible, we will use a Go library to parse the Markdown into an Abstract Syntax Tree (AST). The `goldmark` library is the leading candidate due to its CommonMark compliance, extensibility, and performance.
    *   **Implementation:**
        *   Implement the core Markdown-to-Docs-JSON converter in `internal/drive`. This will involve traversing the AST and building the corresponding Google Docs JSON structure. We will start with basic elements (headings, paragraphs, bold, italic) and expand from there.
2.  **Phase 2: Create New Doc**
    *   Implement the `docs.create` CLI command and MCP tool.
    *   This will be the first real-world test of the converter.
3.  **Phase 3: Add/Replace Tab (Advanced)**
    *   Investigate the feasibility and complexity of the "add tab" and "replace tab" scenarios.
    *   If feasible, implement the `docs.add-tab` and `docs.replace-tab` commands and tools. This will likely be a significant undertaking.
4.  **Phase 4: Documentation and Testing**
    *   Create a new test plan for the Markdown import functionality.
    *   Update the `README.md` to document the new commands.

## 5. Lessons Learned

### Validating Import Methods

Before committing to building a complex parser, we tested a simpler approach based on a suggestion that the Google Drive API could handle the conversion automatically.

*   **Hypothesis:** The `drive.files.create` method could convert a Markdown file to a Google Doc if the source `MimeType` was set to `text/markdown` and the destination `MimeType` was `application/vnd.google-apps.document`.
*   **Test:** We implemented a temporary `test-md-import` command to perform this exact operation.
*   **Result:** The test created a new Google Doc, but the content was the raw, unformatted Markdown text. The Drive API did not perform any conversion of the formatting.
*   **Conclusion:** This experiment definitively proved that there is no built-in, automatic conversion for Markdown in the Drive or Docs APIs. Our initial plan to build a custom "Markdown to Google Docs JSON" converter is the correct and necessary path forward.

### Converter Implementation Strategy

The process of converting a Markdown AST to a series of Google Docs API requests is complex. A naive approach of walking the AST and generating requests on the fly proved to be brittle, especially for handling nested styles and paragraph-level formatting.

*   **Initial Approach:** The first implementation attempted to create `InsertText` and `Update...Style` requests in a single pass of the AST.
*   **Problem:** This led to incorrect styling, such as heading styles being applied to the entire document, because the ranges for the style updates were not being calculated correctly.
*   **Revised Strategy:** A more robust approach is to:
    1.  **Iterate through top-level blocks** of the Markdown document (headings, paragraphs, lists).
    2.  **Process each block independently**, generating the necessary `InsertText` and `Update...Style` requests for that block.
    3.  **Use a reference document:** To accelerate development, we can manually create a Google Doc with the desired formatting for all our target Markdown elements. We can then use the `docs.documents.get` method to inspect the JSON structure of this document. This will serve as a "Rosetta Stone" to guide the implementation of the converter.
