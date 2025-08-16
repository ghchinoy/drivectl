# Manual Test Plan for drivectl (Slides)

This document outlines the manual test cases to verify the functionality of the `drivectl slides` commands.

### Prerequisites

*   Build the latest version of the tool: `go build -o drivectl .`
*   Have a presentation with notes on some slides.

---

### Test Case 1: `slides get` (Text)

*   **Action:** Run `go run . slides get <presentation-id>`
*   **Verification:** Is the text content of the presentation printed to the console?

---

### Test Case 2: `slides get` (PDF)

*   **Action:** Run `go run . slides get <presentation-id> --format pdf -o test.pdf`
*   **Verification:** Is a valid, viewable PDF file named `test.pdf` created?

---

### Test Case 3: `slides get` (Images)

*   **Action:**
    1.  Create a directory: `mkdir test-images`
    2.  Run `go run . slides get <presentation-id> --format png -o test-images`
*   **Verification:** Are the slides of the presentation exported as PNG files to the `test-images` directory?

---

### Test Case 4: `slides create`

*   **Action:** Run `go run . slides create "My New Presentation"`
*   **Verification:** Is a new, blank presentation named "My New Presentation" created in your Google Drive? Is the presentation ID printed to the console?

---

### Test Case 5: `slides create-from` (Markdown)

*   **Action:**
    1.  Create a file `test.md` with the following content:
        ```markdown
        # My Presentation

        This is the subtitle.

        ## Slide 2

        This is the body of the second slide.
        ```
    2.  Run `go run . slides create-from test.md`
*   **Verification:** Is a new presentation created with a title slide and a second slide, with the content from the Markdown file?

---

### Test Case 6: `slides create-from` (Go Slides)

*   **Action:**
    1.  Create a file `test.slides` with the following content:
        ```
        My Presentation
        My Name

        * Slide 2

        This is the body of the second slide.
        ```
    2.  Run `go run . slides create-from test.slides`
*   **Verification:** Is a new presentation created with a title slide and a second slide, with the content from the `.slides` file?

---

### Test Case 7: `slides add`

*   **Action:**
    1.  Create a new presentation: `go run . slides create "My Test Presentation"`
    2.  Copy the presentation ID.
    3.  Run `go run . slides add <presentation-id> --title "My New Slide"`
*   **Verification:** Is a new slide with the title "My New Slide" added to the presentation?
