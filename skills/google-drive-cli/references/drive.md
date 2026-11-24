# Google Drive Operations

Use `drivectl` to interact with files in Google Drive. Always use `-O json` for programmatic parsing.

## Listing Files

To list files, use `drivectl list`. It supports querying with Google Drive search syntax.

**Basic list (default 100 items):**
```bash
drivectl list -O json
```

**List with limit:**
```bash
drivectl list --limit 5 -O json
```

**Search by name:**
```bash
drivectl list -q "name contains 'Project'" -O json
```

**Search for Google Docs specifically:**
```bash
drivectl list -q "mimeType='application/vnd.google-apps.document'" -O json
```

## Describing & Downloading File Content

**Inspect detailed metadata:**
```bash
drivectl describe <file-id> -O json
```

**Download raw file content to stdout (or export Google Docs via `--format md|pdf|txt`):**
```bash
drivectl get <file-id>
drivectl get <google-doc-id> --format md -o notes.md
```

## Uploading & Updating Files

```bash
# Upload a local file to Drive (optional --parent <folder-id>, --name, --mime-type)
drivectl upload ./report.pdf --name "Q3 Report.pdf" -O json

# Upload and convert a local file to native Google Slides, Docs, or Sheets
drivectl upload ./deck.pptx --name "Workshop Deck" --convert slides -O json

# Update an existing Drive or Google Workspace file in place (preserves share URL and permissions)
drivectl upload ./deck.pptx --file-id <existing-file-id> --convert slides -O json
```

## Revisions and Comments

**View the revision history:**
```bash
drivectl revisions <file-id> -O json
```

**View threaded comments:**
```bash
drivectl comments <file-id> -O json
```
