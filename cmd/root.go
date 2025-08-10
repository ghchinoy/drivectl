
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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	secretFile    string
	noBrowserAuth bool
	driveSvc      *drive.Service
	docsSvc       *docs.Service
)

var rootCmd = &cobra.Command{
	Use:   "drivectl",
	Short: "A command-line tool for interacting with the Google Drive API",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if secretFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("could not get user home directory: %w", err)
			}
			secretFile = filepath.Join(home, "secrets", "client_google-drive-api_ghchinoy-genai-blackbelt-fishfooding.json")
		}

		ctx := context.Background()
		client, err := newOAuthClient(ctx, secretFile, noBrowserAuth)
		if err != nil {
			return fmt.Errorf("could not create oauth client: %w", err)
		}

		driveSvc, err = drive.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return fmt.Errorf("could not create drive service: %w", err)
		}

		docsSvc, err = docs.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return fmt.Errorf("could not create docs service: %w", err)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&secretFile, "secret-file", "", "path to your client secrets file (default is ~/secrets/client_google-drive-api_ghchinoy-genai-blackbelt-fishfooding.json)")
	rootCmd.PersistentFlags().BoolVar(&noBrowserAuth, "no-browser-auth", false, "do not open a browser for authentication")
}
