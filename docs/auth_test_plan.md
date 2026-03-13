# Authentication Verification Test Plan

This document outlines the steps to verify the newly implemented robust authentication system, including the `auth login` command, token caching, and auto-refresh logic.

## 1. Initial Setup & First Login

1. **Clear Existing State (if any):**
   ```bash
   rm -rf ~/.config/drivectl/
   ```

2. **Execute Login (Failure expected without secrets):**
   Run the login command without providing a client secret.
   ```bash
   ./drivectl auth login
   ```
   *Expected Result:* An error indicating that the client secret file was not found, prompting the user to provide it via `--secret-file`.

3. **Execute Login (Success):**
   Run the login command with your `client_secret.json` downloaded from Google Cloud Console.
   ```bash
   ./drivectl auth login --secret-file /path/to/your/client_secret.json
   ```
   *Expected Result:*
   * The CLI prints "Copying client secrets..."
   * The CLI opens your default web browser to the Google OAuth consent screen.
   * After consenting, the browser displays "Authentication successful! You can close this browser window."
   * The CLI prints "Login successful! You can now run drivectl commands."

4. **Verify Cache:**
   Check that the config directory was created and contains the secrets and the token.
   ```bash
   ls ~/.config/drivectl/
   ```
   *Expected Result:* Both `client_secret.json` and `token.json` are present.

## 2. Running Commands Without Secrets

1. **Test Standard Command:**
   Run a command that requires authentication, completely omitting the `--secret-file` flag.
   ```bash
   ./drivectl list
   ```
   *Expected Result:* The command executes successfully and lists your Drive files. No warnings about missing secret files should appear.

## 3. Token Auto-Refresh Logic

1. **Simulate Expired Token:**
   Open the cached token file:
   ```bash
   vim ~/.config/drivectl/token.json
   ```
   Find the `"expiry"` field (e.g., `"expiry": "2026-03-13T18:00:00.000000Z"`) and change it to a date in the *past* (e.g., `"expiry": "2024-01-01T00:00:00.000000Z"`). Save the file.

2. **Test Command (Triggers Refresh):**
   Run a command again.
   ```bash
   ./drivectl list
   ```
   *Expected Result:*
   * The command should still succeed.
   * The `golang.org/x/oauth2` library automatically uses the cached `refresh_token` and `client_secret.json` to obtain a new access token.

3. **Verify Updated Token:**
   Check the `token.json` file again.
   ```bash
   cat ~/.config/drivectl/token.json
   ```
   *Expected Result:* The `"expiry"` field should now be updated to a future date.

## 4. Forced Re-Login

1. **Re-Login with Existing Cache:**
   Run the login command again without passing the `--secret-file` flag.
   ```bash
   ./drivectl auth login
   ```
   *Expected Result:*
   * The CLI should *not* complain about a missing secret file (it uses the cached one).
   * It forces the browser flow again (because it explicitly deletes the old token before starting).
   * After completion, a fresh `token.json` is saved.