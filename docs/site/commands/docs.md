---
title: Google Docs
description: Convert Markdown into formatted Google Docs, export Docs back to Markdown, and inspect document tabs.
---

## Create a Google Doc from Markdown (`docs create`)

`drivectl docs create` creates a new Google Doc and optionally converts a local Markdown file into native Google Docs structural elements (headings, bold/italic/strikethrough, inline code, links, lists, and code blocks):

```sh
# Create an empty Google Doc
drivectl docs create "Meeting Notes" -O json

# Convert a local Markdown file into a formatted Google Doc
drivectl docs create "Design Specification" ./docs/DESIGN_PHILOSOPHY.md -O json
```

When `-O json` is passed, `drivectl` returns the new `documentId`, `title`, and `url` (`https://docs.google.com/document/d/<id>/edit`).

## Export a Google Doc to Markdown (`get --format md`)

Use `drivectl get` with `--format md` to parse the Google Docs AST and reconstruct clean Markdown (including headings, inline formatting, links, bulleted/numbered lists, and Markdown tables):

```sh
# Print Markdown to stdout
drivectl get <document-id> --format md

# Save exported Markdown to a file
drivectl get <document-id> --format md -o exported.md
```

You can also export to plain text (`--format txt`) or PDF (`--format pdf -o output.pdf`).

## Inspect document tabs (`docs tabs`)

Google Docs supports hierarchical document tabs. Use `drivectl docs tabs` to inspect top-level and nested child tabs (`tabId`, `title`, `index`, and `childTabs`):

```sh
drivectl docs tabs <document-id> -O json
```

## Full reference

See the [Docs skill reference](../../../skills/google-drive-cli/references/docs.md) in the repository.
