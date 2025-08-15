## Go Projects

When working with Go projects, use the following commands for common tasks:

*   **Build:** `go build`
*   **Test:** `go test ./...`
*   **Run:** `go run main.go`
*   **Dependencies:** `go mod tidy`

## Code Explanation and Architectural Analysis

When asked to explain a concept, feature, or architecture within the codebase, follow this sequence:
  1.  **Identify Key Terms:** Extract the core concepts from the user's query (e.g., "plugin", "authentication").
  2.  **Broad Search:** Use `glob` and `search_file_content` to find all potentially relevant files.
      Search for the key terms, but also for related architectural patterns like `manager`, `service`,
      `config`, and look for `samples` or `tests` which often provide the clearest usage examples.
  3.  **Synthesize Understanding:** Use `read_many_files` to read the identified code and build a holistic understanding.
  4.  **Identify Happy Path and Edge Cases:** Identify the "happy path" (the most common use case) and any "edge cases" (less common use cases or potential failure points).
  5.  **Structure the Explanation:** Present the answer in a structured way:
   *   Start with a high-level summary or analogy.
   *   Detail the core components and their interactions.
   *   Explain the "how" and "why" of the design, including the happy path and edge cases.
   *   Use code snippets for concrete examples.
   *   Discuss pros, cons, and security implications if relevant.
  6.  **Offer to Persist:** For detailed explanations, offer to write the summary to a file for the user's future reference.

## Refactoring

When asked to refactor code, follow this sequence:

1.  **Understand the Goal:** Clarify the user's refactoring goal. Is it to improve readability, performance, maintainability, or something else?
2.  **Analyze the Code:** Use `read_file` and `read_many_files` to understand the code to be refactored. Identify the specific areas that need improvement.
3.  **Check for Tests:** Before making any changes, check for existing test coverage for the code in question. If there are no tests, inform the user and ask if you should write them before proceeding.
4.  **Identify Patterns:** Look for repeating patterns, code smells, or anti-patterns that can be addressed through refactoring.
5.  **Propose a Plan:** Outline the refactoring steps you will take. Explain how the proposed changes will achieve the user's goal.
6.  **Implement in Small Steps:** Apply the refactoring in small, incremental steps. Use the `replace` tool to make the changes.
7.  **Verify at Each Step:** After each step, run the project's build and test commands to ensure that the changes have not introduced any regressions. This is a critical part of the refactoring process.
8.  **Final Verification:** Once all the refactoring steps are complete, run the full suite of build, lint, and test commands to ensure the final code is correct and adheres to the project's standards.

## Verification and Bug Fixing

When a user reports a bug or asks you to verify a piece of functionality, follow this sequence:

1.  **Understand the Issue:** Clarify the user's report. What is the expected behavior, and what is the actual behavior?
2.  **Reproduce the Issue:** Write a test case or a series of steps to reproduce the issue. This will help you to understand the bug and to verify that your fix works.
3.  **Locate the Code:** Use `search_file_content` and `glob` to find the relevant code.
4.  **Analyze the Code:** Read the code to understand why the bug is occurring.
5.  **Propose a Fix:** Outline the changes you will make to fix the bug.
6.  **Implement the Fix:** Apply the fix using the `replace` tool.
7.  **Verify the Fix:** Run the test case you created in step 2 to verify that the bug is fixed.
8.  **Final Verification:** Run the full suite of build, lint, and test commands to ensure that your changes have not introduced any regressions.

## Core Mandates

**Path Construction:** Before using any file system tool (e.g., 'read_file' or 'write_file'), you must construct the full absolute path for the file_path argument. **CRITICAL: Never pass a relative path (e.g., `foo/bar.txt`) or a simple filename (e.g., `baz.md`) to tools requiring a `file_path`.** Always combine the absolute path of the project's root directory with the file's path relative to the root. If a tool fails due to an invalid path, the first step in recovery must be to verify the path's correctness.

**Verify Ambiguous References:** Before acting on a user's request that refers to a file or entity (e.g., "the final report," "the
     test file"), you must first verify which specific file or entity the user means. If the reference is ambiguous or could match multiple
     items, use tools like `list_directory` to inspect the most likely locations. State your intended target back to the user for
     confirmation before proceeding with any action. This prevents errors caused by incorrect assumptions.
