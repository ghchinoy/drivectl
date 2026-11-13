---
title: Google Drive
description: List, search, describe, download, and upload files in Google Drive, and inspect revisions and comments.
---

## List and search files (`list`)

Use `drivectl list` to list files in your Drive or filter them with Google Drive search syntax (`-q`):

```sh
# List the first 100 files (default)
drivectl list

# Limit results and output structured JSON
drivectl list --limit 10 -O json

# Filter by file name or MIME type
drivectl list -q "name contains 'Roadmap' and trashed = false" -O json
drivectl list -q "mimeType='application/vnd.google-apps.document'" -O json
```

## Inspect file metadata (`describe`)

Fetch full metadata (ID, name, MIME type, owners, timestamps, and permissions) for a specific file ID:

```sh
drivectl describe <file-id> -O json
```

## Download and export files (`get`)

Download binary/text files to stdout or a file (`-o`), or export Google Workspace documents into portable formats (`raw`, `txt`, `md`, `pdf`):

```sh
# Stream raw file content to stdout
drivectl get <file-id>

# Export a Google Doc to Markdown or PDF
drivectl get <google-doc-id> --format md -o document.md
drivectl get <google-doc-id> --format pdf -o document.pdf
```

## Upload local files (`upload`)

Upload a local file to Google Drive with an optional target name (`--name`), parent folder (`--parent`), and explicit MIME type (`--mime-type`):

```sh
# Upload a local file to My Drive
drivectl upload ./architecture.pdf --name "Architecture v2.pdf"

# Upload into a specific Drive folder as JSON
drivectl upload ./metrics.csv --parent <folder-id> --mime-type text/csv -O json
```

## Revision history and comments (`revisions`, `comments`)

Inspect who modified a file or read threaded review comments:

```sh
# List file revisions
drivectl revisions <file-id> -O json

# List threaded comments and replies
drivectl comments <file-id> -O json
```

## Full reference

See the [Drive skill reference](../../../skills/google-drive-cli/references/drive.md) in the repository.
