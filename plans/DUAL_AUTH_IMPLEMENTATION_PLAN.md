# Dual Authentication Implementation Plan

This document outlines the plan for adding support for both user-based OAuth 2.0 and service account authentication to the `drivectl` tool.

## 1. Goal

The goal is to provide users with the flexibility to authenticate as either themselves (via OAuth) or as a service account. This will allow for both interactive, user-driven use cases and automated, server-to-server workflows.

## 2. Proposed CLI Changes

A new flag will be introduced to specify the authentication method.

*   `--auth-method <method>`: This flag will accept two values: `user` (the default) and `service-account`.

The existing `--secret-file` flag will be used for both methods. It will point to the `client_secret.json` for the `user` method, and the service account's JSON key file for the `service-account` method.

## 3. Proposed Code Changes

### 3.1. `internal/drive/auth.go`

The `auth.go` file will be refactored to support both authentication flows.

*   **`NewOAuthClient`:** This function will be renamed to `NewUserClient` to better reflect its purpose.
*   **`NewServiceAccountClient`:** A new function will be created to handle service account authentication. It will:
    *   Take the path to the service account key file as an argument.
    *   Use `google.CredentialsFromJSON` to create credentials from the key file.
    *   Return an `*http.Client` that is authenticated as the service account.
*   **`NewClient`:** A new top-level function will be created that acts as a factory. It will:
    *   Take the `auth-method` and `secret-file` as arguments.
    *   Call either `NewUserClient` or `NewServiceAccountClient` based on the selected method.

### 3.2. `mcp/server.go` and `cmd/root.go`

The service initializer functions (`getDriveSvc`, `getDocsSvc`, `getSheetsSvc`) will be updated to use the new `NewClient` factory function.

The `root.go` file will be updated to include the new `--auth-method` flag.

## 4. Updated User Instructions

The `README.md` will be updated with a new "Authentication" section that explains both methods.

*   It will provide clear instructions on how to create both an OAuth client ID and a service account.
*   It will explain how to use the `--auth-method` flag to select the desired authentication method.

## 5. Implementation Phases

1.  **Refactor `auth.go`:**
    *   Rename `NewOAuthClient` to `NewUserClient`.
    *   Implement `NewServiceAccountClient`.
    *   Implement the `NewClient` factory function.
2.  **Update Service Initializers:**
    *   Modify `getDriveSvc`, `getDocsSvc`, and `getSheetsSvc` to use the new `NewClient` factory.
3.  **Update CLI:**
    *   Add the `--auth-method` flag in `cmd/root.go`.
4.  **Update Documentation:**
    *   Update the `README.md` with the new "Authentication" section.
