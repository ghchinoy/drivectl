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

// Package cmd implements the Cobra CLI commands for drivectl.
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/docs/v1"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/drive"
)

// Command group identifiers.
const (
	GroupAuth        = "auth"
	GroupCore        = "core"
	GroupIntegration = "integration"
	GroupAdvanced    = "advanced"
)

var (
	// Version is the current version of the application, injected at build time.
	Version = "dev"
	// OutputFormat specifies the format of the output (e.g., json).
	OutputFormat string
	// noBrowserAuth is a flag to disable opening the browser for authentication.
	noBrowserAuth bool
	// client is the HTTP client used for all API calls.
	client *http.Client
	// driveSvc is the Google Drive service client.
	driveSvc *googledrive.Service
	// docsSvc is the Google Docs service client.
	docsSvc *docs.Service
	// sheetsSvc is the Google Sheets service client.
	sheetsSvc *sheets.Service
)

var rootCmd = &cobra.Command{
	Use:     "drivectl",
	Version: Version,
	Short:   "A CLI for Google Drive and Docs.",
	Long: `drivectl is a powerful command-line tool for interacting with your Google Drive files.
It allows you to list, describe, and download files, with advanced support for
Google Docs, including exporting to multiple formats and accessing individual tabs.`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if cmd.Name() == "login" || cmd.Name() == "auth" || cmd.Name() == "help" {
			return nil
		}
		ctx := cmd.Context()
		var err error
		if client == nil {
			secretFile := viper.GetString("secret-file")
			client, err = drive.NewOAuthClient(ctx, secretFile, noBrowserAuth)
			if err != nil {
				wrapped := fmt.Errorf("could not create oauth client: %w", err)
				return ui.ErrorWithHint(wrapped, "run 'drivectl auth login' to authenticate")
			}
		}

		svcs, err := drive.NewServicesFromClient(ctx, client)
		if err != nil {
			return err
		}
		driveSvc = svcs.Drive
		docsSvc = svcs.Docs
		sheetsSvc = svcs.Sheets
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// ExecuteContext executes the root command with the provided context.
func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

// ExecuteError executes the root command and returns any errors to the caller.
func ExecuteError() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddGroup(&cobra.Group{ID: GroupAuth, Title: "Authentication Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupCore, Title: "Core Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupIntegration, Title: "Integration Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupAdvanced, Title: "Advanced Commands:"})

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("secret-file", "", "path to your client secrets file")
	rootCmd.PersistentFlags().StringVarP(&OutputFormat, "output-format", "O", "", "output format (e.g. json)")
	rootCmd.PersistentFlags().BoolVar(&noBrowserAuth, "no-browser-auth", false, "do not open a browser for authentication")
	rootCmd.PersistentFlags().Bool("mcp", false, "enable MCP server mode over stdio")
	rootCmd.PersistentFlags().String("mcp-http", "", "enable MCP server mode over HTTP at the given address")
	_ = viper.BindPFlag("secret-file", rootCmd.PersistentFlags().Lookup("secret-file"))
	_ = viper.BindEnv("secret-file", "DRIVE_SECRETS")
	_ = viper.BindPFlag("mcp", rootCmd.PersistentFlags().Lookup("mcp"))
	_ = viper.BindPFlag("mcp-http", rootCmd.PersistentFlags().Lookup("mcp-http"))
}

func initConfig() {
	viper.AutomaticEnv()
}
