---
title: Dynamic Discovery & recipes
description: Call arbitrary Google Workspace REST API endpoints via Google Discovery and chain workflows with JSON recipes.
---

## Dynamic Discovery calls (`call`)

When a task is not covered by a native subcommand (such as managing permissions, moving files, or querying a nested API resource), use `drivectl call`:

```sh
drivectl call <service>.<version>.<resource>.<method> --payload '<json>' -O json
```

### How `--payload` routing works

`drivectl` fetches the live Google API Discovery Document (`https://www.googleapis.com/discovery/v1/apis/<service>/<version>/rest`), resolves the target method's `parameters` and `flatPath`/`path`, and maps every key in your JSON `--payload`:

1. **Path parameters** (such as `fileId` or `documentId`) are interpolated and URL-escaped into the endpoint path.
2. **Query parameters** (such as `fields`, `pageSize`, or `supportsAllDrives`) are appended to the URL query string.
3. **Request body fields** (remaining keys for `POST`, `PUT`, or `PATCH` methods) are serialized as the JSON request body.

### Examples

```sh
# Fetch specific fields for a Drive file
drivectl call drive.v3.files.get \
  --payload '{"fileId": "YOUR_FILE_ID", "fields": "id,name,mimeType,owners"}' \
  -O json

# Grant reader permission on a file silently (including Shared Drives)
drivectl call drive.v3.permissions.create \
  --payload '{"fileId": "YOUR_FILE_ID", "sendNotificationEmail": false, "supportsAllDrives": true, "role": "reader", "type": "user", "emailAddress": "teammate@example.com"}' \
  -O json

# Create a new Google Doc via the Docs v1 API
drivectl call docs.v1.documents.create \
  --payload '{"title": "Q4 Architecture Review"}' \
  -O json
```

:::caution[Silent Permission Grants & ACL Safety]
Google Drive API v3 defaults `sendNotificationEmail=true` on `user` and `group` permission grants when omitted, and raw `drive.v3.permissions.create` calls do not inspect existing permissions before writing. Always include `"sendNotificationEmail": false` and `"supportsAllDrives": true` when calling `drive.v3.permissions.create` directly, or use the sister CLI [`share-with`](https://github.com/ghchinoy/share-with) ([documentation](https://ghchinoy.github.io/share-with/)) for transport-enforced silent-only permission grants, `?resourcekey=...` URL parsing, and pre-grant no-downgrade collaborator protection.
:::

## Multi-step recipes (`run`)

Use `drivectl run` to execute a sequence of CLI commands defined in a JSON recipe file (such as [recipes/sample.json](../../../recipes/sample.json)):

```sh
drivectl run recipes/sample.json
```

```json
{
  "name": "Quick Diagnostics",
  "description": "Lists recent documents.",
  "steps": [
    ["list", "--limit", "2"]
  ]
}
```

## Full reference

See the [Discovery skill reference](../../../skills/google-drive-cli/references/discovery.md) in the repository.
