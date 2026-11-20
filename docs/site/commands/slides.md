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
```

Conversion uses Google Drive's `drive.file` scope and sets the target MIME type to `application/vnd.google-apps.presentation`.

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

## Batch-update slides dynamically (`call`)

Because `drivectl` requests the `presentations` OAuth scope, you can also invoke `slides.v1.presentations.batchUpdate` directly via [`drivectl call`](/drivectl/commands/discovery):

```sh
drivectl call slides.v1.presentations.batchUpdate --payload '{
  "presentationId": "<presentation-id>",
  "requests": [
    { "createSlide": { "slideLayoutReference": { "predefinedLayout": "TITLE_AND_BODY" } } }
  ]
}' -O json
```

> **Note:** If you authenticated before Slides write support was added, re-run `drivectl auth login` once so your cached OAuth token includes `https://www.googleapis.com/auth/presentations`.
