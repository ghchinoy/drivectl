# Google Slides Operations

Use `drivectl` to create, inspect, render, and update Google Slides presentations. Always append `-O json` when parsing output programmatically.

## Upload and Convert `.pptx` to Native Google Slides

Convert a local PowerPoint file into a native Google Slides deck (or update an existing deck in place while preserving its share URL and permissions):

```bash
# Upload and convert a new presentation
drivectl upload ./deck.pptx --name "Q4 Architecture Review" --convert slides -O json

# Update an existing Google Slides presentation in place
drivectl upload ./deck.pptx --file-id <presentation-id> --convert slides -O json
```

## Create a Presentation

```bash
drivectl slides create "Q4 Architecture Review" -O json
```

## Render Presentation as Markdown

Fetches slide text, tables, and speaker notes rendered as structured Markdown (or raw Slides API JSON with `-O json`):

```bash
drivectl slides get <presentation-id>
drivectl slides get <presentation-id> -O json
```

## Inspect Slide & Element Object IDs

List each slide's `objectId`, page element `objectId`s, element types, text previews, and `speakerNotesObjectId` before running targeted updates:

```bash
drivectl slides inspect <presentation-id> -O json
drivectl slides inspect <presentation-id> --slide 3 -O json
```

## Replace Text Across Slides

Swap every occurrence of a string across the presentation via `replaceAllText`:

```bash
drivectl slides replace <presentation-id> "OLD_TEXT" "NEW_TEXT" --match-case -O json
```

## Read or Update Speaker Notes

Read or replace speaker notes on a 1-based slide index:

```bash
# Read notes on slide 1
drivectl slides notes <presentation-id> --slide 1 -O json

# Replace notes inline or from a local file
drivectl slides notes <presentation-id> --slide 1 --set "Welcome everyone!" -O json
drivectl slides notes <presentation-id> --slide 2 --file ./slide2-notes.txt -O json
```

## Advanced Batch Updates (`call`)

For complex layout edits, pass a JSON file or stdin to `slides.v1.presentations.batchUpdate`:

```bash
drivectl call slides.v1.presentations.batchUpdate --payload-file ./batch.json -O json
```
