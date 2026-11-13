---
title: Google Sheets
description: Export Google Sheets to CSV, read cell ranges with A1 notation, and update cell values.
---

## Export a sheet to CSV (`sheets get`)

Export a worksheet tab as raw CSV data to stdout:

```sh
# Export the default first sheet or a named sheet as CSV
drivectl sheets get <spreadsheet-id> --sheet "Sheet1"
```

## Read an A1 range (`sheets get-range`)

Fetch cell values from a specific range using A1 notation (`Sheet1!A1:C5` or `--sheet` + `--range`):

```sh
# Human-readable table output
drivectl sheets get-range <spreadsheet-id> --sheet "Sheet1" --range "A1:C5"

# Structured JSON output for scripts and agents
drivectl sheets get-range <spreadsheet-id> --sheet "Sheet1" --range "A1:C5" -O json
```

## Update a cell or range (`sheets update-range`)

Write a value to a cell or top-left anchor of an A1 range using `USER_ENTERED` value input mode (so formulas, numbers, and dates are parsed naturally by Google Sheets):

```sh
drivectl sheets update-range <spreadsheet-id> "Complete" --sheet "Sheet1" --range "B2" -O json
```

## Full reference

See the [Sheets skill reference](../../../skills/google-drive-cli/references/sheets.md) in the repository.
