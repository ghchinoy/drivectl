// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/drive"
)

var (
	authLoginAgentic       bool
	authLoginReadOnly      bool
	authLoginServices      []string
	authLoginReplaceScopes bool
	authLoginPromptConsent bool
	authLoginPrintScopes   bool
	authLogoutLocalOnly    bool
	authResetLocalOnly     bool
)

var authCmd = &cobra.Command{
	Use:     "auth",
	GroupID: GroupAuth,
	Short:   "Authentication commands",
	Long:    `Manage authentication state and OAuth profiles for drivectl.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Google Drive",
	Long: `Login to Google Drive using OAuth 2.0 with PKCE (S256) and encrypted token caching.
Use --readonly and/or --services (drive,docs,sheets,slides) for least-privilege scopes;
subsequent --services logins merge scopes incrementally unless --replace-scopes is passed.
Use --agentic when authenticating with an OAuth client marked "Use this client for an AI-powered agent"
in Google Auth Platform (defaults to the "agent" profile unless --profile is specified).`,
	Example: `  drivectl auth login --secret-file ./client_secret.json
  drivectl auth login --readonly --services drive,docs --secret-file ./client_secret.json
  drivectl auth login --agentic --secret-file ./agent_client_secret.json
  drivectl auth login --prompt-consent`,
	RunE: runAuthLogin,
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	profile := resolveLoginProfile(viper.GetString("profile"), authLoginAgentic)
	existingMeta, _ := drive.LoadProfileMetadata(profile)

	scopes, err := drive.ComputeLoginScopes(existingMeta.Scopes, authLoginServices, authLoginReadOnly, authLoginReplaceScopes)
	if err != nil {
		return ui.ErrorWithHint(err, "Valid services are: drive, docs, sheets, slides.")
	}

	if authLoginPrintScopes {
		return printScopeList("auth login", scopes)
	}

	targetSecretFile, rawSecret, err := prepareProfileSecretFile(profile, viper.GetString("secret-file"))
	if err != nil {
		return err
	}

	_, _ = drive.ClearProfileToken(profile)

	includeGranted := len(authLoginServices) > 0 && !authLoginReplaceScopes && !authLoginReadOnly
	fmt.Printf("Starting OAuth login flow for profile %q (%d scope(s))...\n", profile, len(scopes))
	if _, err := drive.NewOAuthClientWithOptions(cmd.Context(), drive.AuthOptions{
		SecretFile:           targetSecretFile,
		Profile:              profile,
		Agentic:              authLoginAgentic,
		NoBrowserAuth:        noBrowserAuth,
		UserAgent:            fmt.Sprintf("drivectl/%s", Version),
		Scopes:               scopes,
		PromptConsent:        authLoginPromptConsent,
		IncludeGrantedScopes: includeGranted,
	}); err != nil {
		wrapped := fmt.Errorf("login failed: %w", err)
		return ui.ErrorWithHint(wrapped, "Ensure your client_secret.json is valid and the browser flow completed.")
	}

	if err := drive.SaveProfileMetadata(profile, drive.ProfileMetadata{
		Profile:  profile,
		Agentic:  authLoginAgentic,
		ClientID: drive.ExtractClientID(rawSecret),
		Scopes:   scopes,
	}); err != nil {
		return fmt.Errorf("saving profile metadata: %w", err)
	}

	ui.PrintSuccess("Login successful for profile %q! Credentials are encrypted at rest.", profile)
	return nil
}

func prepareProfileSecretFile(profile, secretFile string) (string, []byte, error) {
	targetSecretFile, err := drive.ProfileSecretFile(profile)
	if err != nil {
		return "", nil, err
	}
	if secretFile != "" && secretFile != targetSecretFile {
		fmt.Printf("Copying client secrets from %s to %s...\n", secretFile, targetSecretFile)
		if err := copySecretFile(secretFile, targetSecretFile); err != nil {
			return "", nil, err
		}
	}
	rawSecret, err := os.ReadFile(targetSecretFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil, ui.ErrorWithHint(
				errors.New("client secret file not found"),
				"Provide one via --secret-file on your first login (or set DRIVE_SECRETS).",
			)
		}
		return "", nil, err
	}
	return targetSecretFile, rawSecret, nil
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Inspect authentication status for a profile",
	Example: `  drivectl auth status
  drivectl auth status --profile agent -O json`,
	RunE: func(_ *cobra.Command, _ []string) error {
		profile := viper.GetString("profile")
		status, err := drive.InspectAuthStatus(profile)
		if err != nil {
			return err
		}
		if tf := strings.TrimSpace(viper.GetString("token-file")); tf != "" {
			status.BearerTokenSource = "file:" + tf
		}

		if OutputFormat == "json" {
			data, err := json.MarshalIndent(status, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		printAuthStatus(status)
		return nil
	},
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Force an immediate OAuth token refresh for a profile",
	Example: `  drivectl auth refresh
  drivectl auth refresh --profile agent -O json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		profile := viper.GetString("profile")
		secretFile := viper.GetString("secret-file")
		tok, err := drive.RefreshProfileToken(cmd.Context(), profile, secretFile)
		if err != nil {
			return ui.ErrorWithHint(err, "Run 'drivectl auth login --prompt-consent' if your refresh token expired or is missing.")
		}

		normProfile, _ := drive.NormalizeProfile(profile)
		expiry := ""
		if !tok.Expiry.IsZero() {
			expiry = tok.Expiry.UTC().Format(time.RFC3339)
		}

		if OutputFormat == "json" {
			return printJSON(map[string]any{
				"profile":           normProfile,
				"token_valid":       tok.Valid(),
				"token_expiry":      expiry,
				"has_refresh_token": tok.RefreshToken != "",
				"encrypted":         true,
			})
		}

		ui.PrintSuccess("Refreshed encrypted OAuth token for profile %q (expires %s).", normProfile, expiry)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke and remove the cached OAuth token for a profile",
	Example: `  drivectl auth logout
  drivectl auth logout --profile agent
  drivectl auth logout --local-only`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		profile := viper.GetString("profile")
		normProfile, err := drive.NormalizeProfile(profile)
		if err != nil {
			return err
		}

		tokenPath, err := drive.ProfileTokenCacheFile(normProfile)
		if err != nil {
			return err
		}

		revoked, revokeWarning := revokeProfileTokenBestEffort(cmd.Context(), tokenPath, authLogoutLocalOnly)

		if _, err := drive.ClearProfileToken(normProfile); err != nil {
			return err
		}

		if OutputFormat == "json" {
			return printJSON(map[string]any{
				"profile":        normProfile,
				"token_path":     tokenPath,
				"revoked":        revoked,
				"revoke_warning": revokeWarning,
			})
		}

		if revoked {
			ui.PrintSuccess("Revoked OAuth token and removed cached credentials at %s", tokenPath)
			return nil
		}
		ui.PrintSuccess("Removed cached token at %s", tokenPath)
		return nil
	},
}

var authResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Revoke tokens and remove all cached credentials and metadata for a profile",
	Example: `  drivectl auth reset
  drivectl auth reset --profile agent --local-only`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		profile := viper.GetString("profile")
		resetCtx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		res, err := drive.ResetProfile(resetCtx, profile, authResetLocalOnly)
		if err != nil {
			return err
		}

		if OutputFormat == "json" {
			return printJSON(res)
		}

		if res.RevokeWarning != "" {
			fmt.Fprintf(os.Stderr, "Warning: remote token revocation failed (%s).\n", res.RevokeWarning)
		}
		ui.PrintSuccess("Reset profile %q (%d file(s) removed; revoked=%t).", res.Profile, len(res.RemovedFiles), res.Revoked)
		return nil
	},
}

var authScopesCmd = &cobra.Command{
	Use:   "scopes [command...]",
	Short: "Show minimal OAuth scopes required for a drivectl command",
	Example: `  drivectl auth scopes list
  drivectl auth scopes upload
  drivectl auth scopes slides notes -O json`,
	RunE: func(_ *cobra.Command, args []string) error {
		target := strings.Join(args, " ")
		return printScopeList(target, requiredScopesForCommandPath(target))
	},
}

func revokeProfileTokenBestEffort(ctx context.Context, tokenPath string, localOnly bool) (bool, string) {
	if localOnly {
		return false, ""
	}
	tok, err := drive.TokenFromFile(tokenPath)
	if err != nil {
		return false, ""
	}
	revokeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := drive.RevokeToken(revokeCtx, nil, tok); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: remote token revocation failed (%v); removing local cache anyway.\n", err)
		return false, err.Error()
	}
	return true, ""
}

func resolveLoginProfile(explicitProfile string, agentic bool) string {
	trimmed := strings.TrimSpace(explicitProfile)
	if trimmed != "" {
		return trimmed
	}
	if agentic {
		return drive.AgentProfile
	}
	return drive.DefaultProfile
}

func printAuthStatus(s drive.AuthStatus) {
	fmt.Printf("Profile:       %s\n", s.Profile)
	fmt.Printf("Agentic:       %t\n", s.Agentic)
	fmt.Printf("Config Dir:    %s\n", s.ConfigDir)
	if s.BearerTokenSource != "" {
		fmt.Printf("Bearer Token:  active (%s)\n", s.BearerTokenSource)
	}
	fmt.Printf("Client Secret: %s (exists=%t)\n", s.SecretFile, s.SecretExists)
	if s.ClientID != "" {
		fmt.Printf("Client ID:     %s\n", s.ClientID)
	}
	fmt.Printf("Token File:    %s (exists=%t, encrypted=%t)\n", s.Token.Path, s.Token.Exists, s.Token.Encrypted)
	if s.Token.Exists {
		fmt.Printf("Token Valid:   %t (has_refresh=%t)\n", s.TokenValid, s.HasRefresh)
		if s.TokenExpiry != "" {
			fmt.Printf("Token Expiry:  %s\n", s.TokenExpiry)
		}
	}
	if len(s.Scopes) > 0 {
		fmt.Printf("Scopes:        %s\n", strings.Join(s.Scopes, ", "))
	}
}

func copySecretFile(srcPath, dstPath string) error {
	source, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source secret file %q: %w", srcPath, err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create target secret file %q: %w", dstPath, err)
	}

	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to copy secret file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close target secret file %q: %w", dstPath, closeErr)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(authCmd)
	authLoginCmd.Flags().BoolVar(&authLoginAgentic, "agentic", false, `mark this profile as using a Google Auth Platform "AI-powered agent" OAuth client (defaults --profile to "agent")`)
	authLoginCmd.Flags().BoolVar(&authLoginReadOnly, "readonly", false, "request read-only OAuth scopes only")
	authLoginCmd.Flags().StringSliceVar(&authLoginServices, "services", nil, "comma-separated services to authorize (drive,docs,sheets,slides); merges incrementally with existing profile scopes")
	authLoginCmd.Flags().BoolVar(&authLoginReplaceScopes, "replace-scopes", false, "replace previously cached profile scopes instead of merging incrementally")
	authLoginCmd.Flags().BoolVar(&authLoginPromptConsent, "prompt-consent", false, "force Google's consent screen (prompt=consent) to ensure a fresh refresh token")
	authLoginCmd.Flags().BoolVar(&authLoginPrintScopes, "print-scopes", false, "print the resolved OAuth scopes and exit without logging in")
	authLogoutCmd.Flags().BoolVar(&authLogoutLocalOnly, "local-only", false, "remove the local cached token without calling Google's revocation endpoint")
	authResetCmd.Flags().BoolVar(&authResetLocalOnly, "local-only", false, "reset local profile files without calling Google's revocation endpoint")
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authRefreshCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authResetCmd)
	authCmd.AddCommand(authScopesCmd)
}
