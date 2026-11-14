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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/drive"
)

var authCmd = &cobra.Command{
	Use:     "auth",
	GroupID: GroupAuth,
	Short:   "Authentication commands",
	Long:    `Manage authentication state for drivectl.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Google Drive",
	Long: `Login to Google Drive using OAuth2.
This command will open your browser to authenticate with Google.
It will save the credentials to a local configuration directory for future use.`,
	Example: `  drivectl auth login --secret-file ./client_secret.json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		secretFile := viper.GetString("secret-file")

		configDir, err := drive.ConfigDir()
		if err != nil {
			return err
		}
		targetSecretFile := filepath.Join(configDir, "client_secret.json")

		if secretFile != "" && secretFile != targetSecretFile {
			fmt.Printf("Copying client secrets from %s to %s...\n", secretFile, targetSecretFile)
			if err := copySecretFile(secretFile, targetSecretFile); err != nil {
				return err
			}
		}

		if _, err := os.Stat(targetSecretFile); os.IsNotExist(err) {
			return ui.ErrorWithHint(
				errors.New("client secret file not found"),
				"Provide one via --secret-file on your first login.",
			)
		}

		tokenFile, err := drive.TokenCacheFile()
		if err == nil {
			_ = os.Remove(tokenFile)
		}

		fmt.Println("Starting OAuth login flow...")
		if _, err := drive.NewOAuthClient(cmd.Context(), targetSecretFile, noBrowserAuth); err != nil {
			wrapped := fmt.Errorf("login failed: %w", err)
			return ui.ErrorWithHint(wrapped, "Ensure your client_secret.json is valid and the browser flow completed.")
		}

		ui.PrintSuccess("Login successful! You can now run drivectl commands.")
		return nil
	},
}

func copySecretFile(srcPath, dstPath string) error {
	source, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source secret file: %w", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create target secret file: %w", err)
	}

	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to copy secret file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close target secret file: %w", closeErr)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
}
