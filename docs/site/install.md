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

Before authenticating for the first time, create a **Desktop app** OAuth 2.0 client ID in the Google Cloud Console.

:::caution[Use a Desktop OAuth 2.0 client JSON]
`drivectl` requires an **OAuth 2.0 Client ID (Desktop app)** JSON file whose top-level key is `{"installed": {...}}`. Do **not** use:

- a **Service Account** JSON key (`{"type": "service_account", ...}` — not currently supported),
- an **API key**, or
- a **Web application** OAuth client (`{"web": ...}`), which rejects loopback redirects unless `http://localhost:8080` is manually registered.
:::

1. Open the [Google Cloud Console](https://console.cloud.google.com/) and select or create a project.
2. Navigate to **APIs & Services > Library** and enable:
   - **Google Drive API** (`drive.googleapis.com`)
   - **Google Docs API** (`docs.googleapis.com`)
   - **Google Sheets API** (`sheets.googleapis.com`)
   - **Google Slides API** (`slides.googleapis.com`)
3. Configure the OAuth consent screen under **Google Auth Platform** (or **APIs & Services > OAuth consent screen**):
   - Under **Branding**, set an app name and support email.
   - Under **Audience**, choose **Internal** (if you are in a Google Workspace organization and only org users will sign in) or **External** (for personal Gmail accounts or cross-org use).
   - If **External** stays in **Testing** publishing status, open **Audience > Test users** and add every Google email address that will sign in — otherwise login fails with `Error 403: access_denied`. *(Google expires refresh tokens for External apps in Testing mode after 7 days; re-run `drivectl auth login` when needed.)*
4. Open **Google Auth Platform > Clients** (or **APIs & Services > Credentials**), select **Create Client** / **Create Credentials > OAuth client ID**, and choose **Desktop app**.
5. Download the JSON file (verify it starts with `{"installed": ...}`) and save it as `client_secret.json`.

## Next step

Continue to [Getting started](/drivectl/getting-started) to log in once with `drivectl auth login` and run your first commands.
