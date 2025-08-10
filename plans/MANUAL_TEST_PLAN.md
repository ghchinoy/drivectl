# Manual Test Plan for drivectl

This document outlines the manual test cases to verify the functionality of the `drivectl` tool. Check off items as you complete them.

### Prerequisites

- [x] Build the latest version of the tool: `go build -o drivectl .`
- [x] In your Google Drive, create a test Google Doc named `TestDoc`. Add some text to it.
- [x] In your Google Drive, upload a non-Google Doc file (e.g., a small image named `test-image.png`).

---

### Test Case 1: First-Time Authentication (Browser Flow)

- [x] **Action:** Delete the old token file: `rm ~/.config/drivectl/token.json`
- [x] **Action:** Run the list command: `./drivectl list`
- [x] **Verification:**
    - [x] Did your web browser automatically open to the Google authentication screen?
    - [x] After approval, did the web page show a success message?
    - [x] Did the command complete successfully and print a list of your Drive files?
    - [x] Does the `~/.config/drivectl/token.json` file now exist?

---

### Test Case 2: Subsequent Runs (Token Usage)

- [x] **Action:** Run the list command again: `./drivectl list`
- [x] **Verification:** Did the command immediately list your files without opening a browser?

---

### Test Case 3: `list` Command Functionality

- [x] **Action:** Run with a limit: `./drivectl list --limit 5`
- [x] **Verification:** Are exactly 5 files (or fewer, if you have less than 5) listed?
- [x] **Action:** Run with a query: `./drivectl list -q "name = 'TestDoc'"`
- [x] **Verification:** Is only the file named `TestDoc` listed?

---

### Test Case 4: `describe` Command Functionality

- [x] **Action:**
    1. Copy the file ID for `TestDoc` from the `list` command output.
    2. Run the describe command: `./drivectl describe <TestDoc-file-id>`
- [x] **Verification:** Is a detailed JSON object with the file's metadata printed to the console?

---

### Test Case 5: `get` Command (Google Doc)

- [x] **Action:**
    1. Get the file ID for `TestDoc`.
    2. Run to print to console: `./drivectl get <TestDoc-file-id>`
    3. Run to save to a file: `./drivectl get <TestDoc-file-id> -o test-doc.txt`
- [x] **Verification:**
    - [x] Is the plain text content of `TestDoc` printed to the console?
    - [x] Is a new file named `test-doc.txt` created with the correct content?

---

### Test Case 6: `get` Command (Regular File)

- [x] **Action:**
    1. Get the file ID for your `test-image.png`.
    2. Run to save the file: `./drivectl get <test-image-file-id> -o downloaded-image.png`
- [x] **Verification:**
    - [x] Is a new file named `downloaded-image.png` created?
    - [x] Is the downloaded file identical to the original?

---

### Test Case 7: Manual Authentication Flow (`--no-browser-auth`)

- [x] **Action:**
    1. Delete the token file: `rm ~/.config/drivectl/token.json`
    2. Run with the no-browser flag: `./drivectl list --no-browser-auth`
    3. Follow the printed URL, authorize, and copy the code from the redirect URL.
    4. Paste the code into the waiting terminal.
- [x] **Verification:**
    - [x] Did authentication succeed?
    - [x] Did the command print a list of your Drive files?
    - [x] Is a new `token.json` file created?

---

### Test Case 8: `get` Command (Format Flag)

- [x] **Action:**
    1. Get the file ID for `TestDoc`.
    2. Run to export as PDF: `./drivectl get <TestDoc-file-id> --format pdf -o test.pdf`
    3. Run to export as a Word document: `./drivectl get <TestDoc-file-id> --format docx -o test.docx`
- [x] **Verification:**
    - [x] Is a valid, viewable PDF file named `test.pdf` created?
    - [x] Is a valid, viewable Word document named `test.docx` created?
