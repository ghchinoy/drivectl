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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/drive"
)

var authLoginAgentic bool

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
Use --agentic when authenticating with an OAuth client marked "Use this client for an AI-powered agent"
in Google Auth Platform (defaults to the "agent" profile unless --profile is specified).`,
	Example: `  drivectl auth login --secret-file ./client_secret.json
  drivectl auth login --agentic --secret-file ./agent_client_secret.json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		profile := resolveLoginProfile(viper.GetString("profile"), authLoginAgentic)
		secretFile := viper.GetString("secret-file")

		targetSecretFile, err := drive.ProfileSecretFile(profile)
		if err != nil {
			return err
		}

		if secretFile != "" && secretFile != targetSecretFile {
			fmt.Printf("Copying client secrets from %s to %s...\n", secretFile, targetSecretFile)
			if err := copySecretFile(secretFile, targetSecretFile); err != nil {
				return err
			}
		}

		rawSecret, err := os.ReadFile(targetSecretFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return ui.ErrorWithHint(
					errors.New("client secret file not found"),
					"Provide one via --secret-file on your first login.",
				)
			}
			return err
		}

		_, _ = drive.ClearProfileToken(profile)

		fmt.Printf("Starting OAuth login flow for profile %q...\n", profile)
		if _, err := drive.NewOAuthClientWithOptions(cmd.Context(), drive.AuthOptions{
			SecretFile:    targetSecretFile,
			Profile:       profile,
			Agentic:       authLoginAgentic,
			NoBrowserAuth: noBrowserAuth,
			UserAgent:     fmt.Sprintf("drivectl/%s", Version),
		}); err != nil {
			wrapped := fmt.Errorf("login failed: %w", err)
			return ui.ErrorWithHint(wrapped, "Ensure your client_secret.json is valid and the browser flow completed.")
		}

		if err := drive.SaveProfileMetadata(profile, drive.ProfileMetadata{
			Profile:  profile,
			Agentic:  authLoginAgentic,
			ClientID: drive.ExtractClientID(rawSecret),
			Scopes:   drive.DefaultOAuthScopes(),
		}); err != nil {
			return fmt.Errorf("saving profile metadata: %w", err)
		}

		ui.PrintSuccess("Login successful for profile %q! Credentials are encrypted at rest.", profile)
		return nil
	},
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

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove cached OAuth token for a profile",
	Example: `  drivectl auth logout
  drivectl auth logout --profile agent`,
	RunE: func(_ *cobra.Command, _ []string) error {
		profile := viper.GetString("profile")
		tokenPath, err := drive.ClearProfileToken(profile)
		if err != nil {
			return err
		}
		ui.PrintSuccess("Removed cached token at %s", tokenPath)
		return nil
	},
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
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}
