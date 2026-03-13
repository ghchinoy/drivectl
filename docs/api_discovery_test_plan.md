# API Discovery Verification Test Plan

This document outlines the steps to verify the newly implemented `drivectl call` command and the dynamic Google API Discovery Document fetching mechanism.

## Prerequisites
* Ensure you have successfully logged in using `drivectl auth login` and have a valid token cache.

## 1. Verify Discovery Document Caching

1. **Clear Existing Discovery Cache:**
   ```bash
   rm -rf ~/.config/drivectl/discovery/
   ```
2. **Execute a Dynamic Call:**
   Run a simple `drive.v3.files.list` command.
   ```bash
   ./drivectl call drive.v3.files.list --payload '{"pageSize": 2}'
   ```
   *Expected Result:*
   * The command succeeds and returns a JSON payload containing up to 2 files.
   * Because the cache was cleared, it dynamically downloaded the discovery document from `https://www.googleapis.com/discovery/v1/apis/drive/v3/rest`.

3. **Verify the Cache File:**
   ```bash
   ls ~/.config/drivectl/discovery/
   ```
   *Expected Result:* You should see a file named `drive_v3.json`.

## 2. Test Path Parameter Resolution

The `call` command must correctly extract parameters meant for the URL path from the JSON payload.

1. **Find a File ID:**
   Pick any valid File ID from the output of the previous step.
2. **Execute `drive.v3.files.get`:**
   Pass the File ID in the payload. Note how `fileId` is required for the path according to the API schema.
   ```bash
   ./drivectl call drive.v3.files.get --payload '{"fileId": "YOUR_FILE_ID", "fields": "name,mimeType"}'
   ```
   *Expected Result:*
   * The CLI successfully replaces `{fileId}` in the URL.
   * The CLI outputs the metadata of the file.

## 3. Test Missing Required Parameters

1. **Omit a Required Parameter:**
   Try to fetch a file without passing its `fileId`.
   ```bash
   ./drivectl call drive.v3.files.get --payload '{"fields": "name"}'
   ```
   *Expected Result:* The CLI should fail *locally* before making a network request, printing an error like: `missing required path parameter: fileId`.

## 4. Test Different APIs (Fallback Discovery URLs)

Some newer Google APIs use a different discovery endpoint format (`$discovery/rest`). The CLI is designed to fall back to this format if the standard one fails.

1. **Test the Docs API:**
   ```bash
   ./drivectl call docs.v1.documents.create --payload '{"title": "Drivectl Discovery Test"}'
   ```
   *Expected Result:* 
   * A new Google Doc is created.
   * You should see `docs_v1.json` added to your `~/.config/drivectl/discovery/` cache.
   * The terminal outputs the JSON response of the newly created document.