---
title: Install drivectl
description: Install the drivectl binary, configure Google OAuth credentials, and optional agent integrations.
---

## Prebuilt binaries (macOS & Linux)

Download and verify the latest release binary into your current directory using the [bundled installer script](../../skills/google-drive-cli/scripts/install.sh):

```sh
curl -sL https://raw.githubusercontent.com/ghchinoy/drivectl/main/skills/google-drive-cli/scripts/install.sh | bash
```

The script detects your operating system (`Darwin` or `Linux`) and architecture (`x86_64` or `arm64`), downloads the release archive from GitHub Releases, verifies its SHA-256 checksum, and extracts `drivectl`.

## Go install

If you have **Go 1.26+** installed:

```sh
go install github.com/ghchinoy/drivectl@latest
```

## Build from source

Clone the repository and compile the binary locally:

```sh
git clone https://github.com/ghchinoy/drivectl.git
cd drivectl
go build -o drivectl .
```

## Configure a Google OAuth client secret

Before authenticating for the first time, create a Desktop OAuth 2.0 client ID in the Google Cloud Console:

1. Open the [Google Cloud Console](https://console.cloud.google.com/) and select or create a project.
2. Navigate to **APIs & Services > Library** and enable:
   - **Google Drive API** (`drive.googleapis.com`)
   - **Google Docs API** (`docs.googleapis.com`)
   - **Google Sheets API** (`sheets.googleapis.com`)
3. Open **APIs & Services > Credentials**, select **Create Credentials > OAuth client ID**, and choose **Desktop app**.
4. Download the JSON file and save it as `client_secret.json`.

## Next step

Continue to [Getting started](/getting-started) to log in once with `drivectl auth login` and run your first commands.
