---
title: Google Slides
description: Inspect, create, and upload-convert Google Slides presentations from the command line.
---

## Upload and convert a `.pptx` deck to Google Slides

The fastest way to publish a richly styled presentation into Google Drive as a native Google Slides deck is `drivectl upload --convert slides`:

```sh
# Upload a local PowerPoint file and convert it to native Google Slides
drivectl upload ./workshop-deck.pptx --name "DevFest 2026 Workshop" --convert slides

# Upload into a specific folder and emit JSON (includes id and webViewLink)
drivectl upload ./workshop-deck.pptx --parent <folder-id> --convert slides -O json

# Update an existing Google Slides deck in place (preserves share link and permissions)
drivectl upload ./workshop-deck.pptx --file-id <existing-id> --convert slides
```

Conversion uses Google Drive's `drive.file` scope and sets the target MIME type to `application/vnd.google-apps.presentation`. When `--name` is omitted with `--convert`, `drivectl` strips the local file extension automatically.

## Create a presentation via the Slides API (`slides create`)

Create a blank Google Slides presentation directly via the Slides API:

```sh
# Create a presentation and print its ID and edit URL
drivectl slides create "DevFest 2026 Workshop"

# Emit full presentation JSON
drivectl slides create "DevFest 2026 Workshop" -O json
```

## Render a presentation as Markdown (`slides get`)

Fetch a presentation and render its slide text, tables, and speaker notes as structured Markdown, or pass `-O json` for the full Slides API response:

```sh
# Render slides, tables, and speaker notes as Markdown
drivectl slides get <presentation-id>

# Retrieve the full Slides API JSON structure
drivectl slides get <presentation-id> -O json
```

## Inspect slide and element IDs (`slides inspect`)

List every slide's `objectId`, page element `objectId`s, element types, text previews, and `speakerNotesObjectId`—ideal before crafting targeted `batchUpdate` edits:

```sh
# Inspect all slides
drivectl slides inspect <presentation-id>

# Inspect a single slide as structured JSON
drivectl slides inspect <presentation-id> --slide 3 -O json
```

## Replace text across slides (`slides replace`)

Swap every occurrence of a string across a presentation using the Slides API `replaceAllText` request:

```sh
drivectl slides replace <presentation-id> "Speaker Name" "G. Hussain Chinoy" --match-case
```

## Read or update speaker notes (`slides notes`)

Read or replace speaker notes on a 1-based slide index in a single command:

```sh
# Read speaker notes on slide 1
drivectl slides notes <presentation-id> --slide 1

# Update speaker notes inline or from a file
drivectl slides notes <presentation-id> --slide 1 --set "Welcome everyone!"
drivectl slides notes <presentation-id> --slide 2 --file ./notes.txt
```

## Batch-update slides dynamically (`call`)

Because `drivectl` requests the `presentations` OAuth scope, you can also invoke `slides.v1.presentations.batchUpdate` directly via [`drivectl call`](/drivectl/commands/discovery) using `--payload` or `--payload-file`:

```sh
drivectl call slides.v1.presentations.batchUpdate --payload-file ./batch.json
```

> **Note:** If you authenticated before Slides write support was added, re-run `drivectl auth login` once so your cached OAuth token includes `https://www.googleapis.com/auth/presentations`.
