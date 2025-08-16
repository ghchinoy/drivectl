# Google Slides Implementation Plan

## 1. Overview

This document outlines the plan to integrate Google Slides functionality into the `drivectl` command-line tool. The goal is to provide a comprehensive set of features for both reading and creating presentations, aligning with the tool's existing support for Google Docs and Sheets. All new functionality will also be exposed through the MCP server to enable automation.

## 2. Command Structure

A new `slides` subcommand will be added to house all Slides-related operations.

### 2.1. Reading and Exporting Presentations

*   **`drivectl slides get <presentation-id>`**: Retrieves a presentation.
    *   `--format <pdf|png|jpg>`: Specifies the output format.
        *   `pdf`: Exports the entire presentation as a PDF.
        *   `png|jpg`: Exports each slide as an individual image file.
    *   `-o, --output <file|dir>`: Specifies the output file (for PDF) or directory (for images).

*   **`drivectl slides notes <presentation-id>`**: Extracts speaker notes.
    *   `--slide-number <number>`: (Optional) Specifies a single slide to get notes from.
    *   `--format <txt|md>`: (Optional) Specifies the output format for the notes. Defaults to plain text.
    *   `-o, --output <file>`: (Optional) Specifies an output file.

### 2.2. Creating and Modifying Presentations

*   **`drivectl slides create <title>`**: Creates a new, blank presentation.
    *   **Returns**: The ID of the newly created presentation.

*   **`drivectl slides create-from <file>`**: Creates a new presentation from a source file.
    *   `--type <markdown|go-slides>`: Specifies the source file type. If not provided, the tool will infer from the file extension (`.md`, `.slides`).
    *   `--title <title>`: (Optional) Overrides the title defined in the source file.

*   **`drivectl slides add <presentation-id>`**: Adds a new slide to an existing presentation.
    *   `--title <title>`: The title for the new slide.
    *   `--layout <layout>`: The layout for the new slide (e.g., `TITLE_AND_BODY`, `BLANK`).

## 3. Feature Details

### 3.1. `create-from` Parsing Logic

*   **Markdown (`.md`)**:
    *   The first `H1` will be the presentation title.
    *   Each `H2` will create a new slide with the heading as the slide title.
    *   Bulleted lists under an `H2` will become bullet points on the corresponding slide.
    *   Local image paths will be resolved, the images uploaded, and then inserted into the slides.
*   **Go Slides (`.slides`)**:
    *   The tool will use the `golang.org/x/tools/present` package to parse the `.slides` file.
    *   The presentation structure (title, slides, content) will be mapped to the Google Slides API.

### 3.2. MCP Server Integration

All new `slides` subcommands and their flags will be exposed as tools on the MCP server, allowing for programmatic creation and management of presentations.

## 4. Task Checklist

- [x] **Foundation**
    - [x] Add Slides API scope to authentication.
    - [x] Create `cmd/slides.go` and the main `slides` subcommand.
- [x] **Reading Features**
    - [x] Implement `drivectl slides get`.
    - [x] Add PDF export (`--format pdf`).
    - [x] Add image export (`--format png|jpg`).
- [ ] **Creation Features**
    - [ ] Implement `drivectl slides create`.
    - [ ] Implement `drivectl slides create-from`.
    - [ ] Improve Markdown to Slides Formatting.
    - [ ] Set slide background image.
    - [ ] Add Markdown parsing logic.
    - [ ] Add Go `.slides` parsing logic.
    - [ ] Implement `drivectl slides add`.
- [x] **MCP Integration**
    - [x] Expose `slides get` command on the MCP server.
- [ ] **Documentation**
    - [ ] Update `README.md` with examples for all new `slides` commands.
    - [ ] Add a `MANUAL_TEST_PLAN.md` for slides.

## 5. Review and Lessons Learned

### Notes Retrieval Issue (August 15, 2025)

During the implementation of the `drivectl slides notes` command, we encountered a significant issue where the Google Slides API would not return the `notesProperties` for the slides in a specific presentation, even though the presentation was known to have speaker notes. This prevented the tool from being able to access the speaker notes.

**Debugging Steps Taken:**

*   Verified that the user had the correct permissions and that the correct OAuth scopes were being used.
*   Attempted to fetch the presentation with various `fields` masks, including no mask, `slides`, `slides(notesProperties)`, `slides(objectId,notesProperties)`, `slides(objectId,notesProperties/speakerNotesObjectId)`, and `*`. In all cases, the `notesProperties` field was `nil` in the API response.
*   Attempted to fetch each slide individually using `presentations.pages.get`. The `notesProperties` field was still `nil`.

**Conclusion:**

At this time, we have exhausted all known methods for reading the speaker notes for this particular presentation via the API. The API is not providing the necessary information to access the notes. This issue has been tabled for now, and we will revisit it at a later date. It is possible that creating a new presentation with notes via the API will provide more insight into how the notes are structured and how to retrieve them.

### Image Insertion Issue (August 15, 2025)

During the implementation of the `drivectl slides add-image` command, we encountered several issues that prevented us from successfully adding an image to a slide:

*   **Data URI Limit:** The initial approach of using a data URI failed because the generated URI for the image exceeded the 2K byte limit for the URL field in the Slides API's `CreateImageRequest`.
*   **Private `webContentLink`:** Using the `webContentLink` of a private image in Google Drive also failed. The Slides API was unable to access the image, even though the user was authenticated.
*   **Public `webContentLink`:** Attempting to make the image public in Google Drive and then using the `webContentLink` failed with a `403 Forbidden` error, indicating that the user's account does not have the necessary permissions to make files public.

**Conclusion:**

At this time, we have been unable to find a reliable way to add an image to a slide given the limitations of the Slides API and the user's permissions. This feature has been moved to the "Revisit Later" section of the plan.

## 6. Revisit Later

- [ ] **Notes Retrieval**
    - [ ] Re-investigate the issue with reading speaker notes.
    - [ ] Implement `drivectl slides notes`.
    - [ ] Add text and markdown export for notes (`--format txt|md`).