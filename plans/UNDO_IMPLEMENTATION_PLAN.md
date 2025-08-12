# Undo Feature Implementation Plan

This document outlines the plan for implementing an undo feature for sheet write operations in the `drivectl` tool.

## 1. Concept

The core idea is to create a simple, single-level undo command that can revert the most recent write operation made by `drivectl`.

## 2. Proposed Mechanism

1.  **Read Before Writing:** Before any `sheets update-range` operation is executed, the tool would first perform a `get-range` on the exact same range that is about to be modified.
2.  **Cache Original Data:** The original data retrieved in the previous step would be saved to a temporary cache file (e.g., `~/.config/drivectl/undo_buffer.json`). This file would store the spreadsheet ID, the sheet name, the range, and the original values. This file would be overwritten on every new write operation.
3.  **Implement `undo` Command:** We would introduce a new command: `drivectl sheets undo`.
4.  **"Undo" Logic:** When `drivectl sheets undo` is run, it would:
    a.  Read the data from the `undo_buffer.json` file.
    b.  If the file exists and contains data, it would perform a `sheets update-range` operation using the cached information to restore the previous state.
    c.  After a successful undo, the `undo_buffer.json` file would be cleared to prevent accidental repeated undos.

## 3. Limitations of this Approach

*   **Single-Level Undo:** This design only allows undoing the single most recent write operation.
*   **Local to User:** The undo buffer is stored locally on your machine. It cannot undo changes made by other users or by you from a different computer.
*   **Not a True Transaction:** This is not a transactional rollback in a database sense. If other changes have been made to the sheet by other means since the `drivectl` write, an undo could have unintended consequences.