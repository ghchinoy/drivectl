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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/docs/v1"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"

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
	// showScopes prints the minimal OAuth scopes required by a command without executing it.
	showScopes bool
	// savedRunE / savedRun temporarily hold a command's handler when --show-scopes intercepts execution.
	savedRunE func(*cobra.Command, []string) error
	savedRun  func(*cobra.Command, []string)
	// client is the HTTP client used for all API calls.
	client *http.Client
	// driveSvc is the Google Drive service client.
	driveSvc *googledrive.Service
	// docsSvc is the Google Docs service client.
	docsSvc *docs.Service
	// sheetsSvc is the Google Sheets service client.
	sheetsSvc *sheets.Service
	// slidesSvc is the Google Slides service client.
	slidesSvc *slides.Service
)

var rootCmd = &cobra.Command{
	Use:     "drivectl",
	Version: Version,
	Short:   "A CLI for Google Drive and Docs.",
	Long: `drivectl is a powerful command-line tool for interacting with your Google Drive files.
It allows you to list, describe, and download files, with advanced support for
Google Docs, including exporting to multiple formats and accessing individual tabs.`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if showScopes {
			if err := printCommandScopes(cmd); err != nil {
				return err
			}
			savedRunE = cmd.RunE
			savedRun = cmd.Run
			cmd.RunE = func(*cobra.Command, []string) error { return nil }
			cmd.Run = nil
			return nil
		}
		if isAuthBypassCommand(cmd) {
			return nil
		}
		ctx := cmd.Context()
		var err error
		if client == nil {
			client, err = initAuthenticatedClient(ctx, cmd)
			if err != nil {
				return err
			}
		}

		svcs, err := drive.NewServicesFromClient(ctx, client)
		if err != nil {
			return err
		}
		driveSvc = svcs.Drive
		docsSvc = svcs.Docs
		sheetsSvc = svcs.Sheets
		slidesSvc = svcs.Slides
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, _ []string) {
		if savedRunE != nil || savedRun != nil {
			cmd.RunE = savedRunE
			cmd.Run = savedRun
			savedRunE = nil
			savedRun = nil
		}
	},
}

func initAuthenticatedClient(ctx context.Context, cmd *cobra.Command) (*http.Client, error) {
	userAgent := fmt.Sprintf("drivectl/%s", Version)
	bearerToken, _, err := drive.ResolveAccessToken(viper.GetString("token-file"), cmd.InOrStdin())
	if err != nil {
		return nil, ui.ErrorWithHint(err, "Check DRIVECTL_TOKEN or --token-file.")
	}
	if bearerToken != "" {
		return drive.WrapHTTPClientUserAgent(drive.NewClientFromBearerToken(ctx, bearerToken), userAgent), nil
	}

	profile := viper.GetString("profile")
	secretFile := viper.GetString("secret-file")
	httpClient, err := drive.NewOAuthClientWithOptions(ctx, drive.AuthOptions{
		SecretFile:    secretFile,
		Profile:       profile,
		NoBrowserAuth: noBrowserAuth,
		UserAgent:     userAgent,
	})
	if err != nil {
		wrapped := fmt.Errorf("could not create oauth client: %w", err)
		return nil, ui.ErrorWithHint(wrapped, "run 'drivectl auth login' to authenticate (or set DRIVECTL_TOKEN)")
	}
	return httpClient, nil
}

func isAuthBypassCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "auth", "help", "completion":
			return true
		}
	}
	return false
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	ui.HandleError(rootCmd.Execute())
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
	rootCmd.PersistentFlags().String("token-file", "", "path to a file containing a pre-obtained OAuth 2.0 access token (or '-' for stdin; alternative to DRIVECTL_TOKEN)")
	rootCmd.PersistentFlags().String("profile", "", "authentication profile name (default: \"default\"; use \"agent\" for AI agent OAuth clients)")
	rootCmd.PersistentFlags().StringVarP(&OutputFormat, "output-format", "O", "", "output format (e.g. json)")
	rootCmd.PersistentFlags().BoolVar(&noBrowserAuth, "no-browser-auth", false, "do not open a browser for authentication")
	rootCmd.PersistentFlags().BoolVar(&showScopes, "show-scopes", false, "print the minimal OAuth scopes required for the command and exit")
	rootCmd.PersistentFlags().Bool("mcp", false, "enable MCP server mode over stdio")
	rootCmd.PersistentFlags().String("mcp-http", "", "enable MCP server mode over HTTP at the given address (defaults to 127.0.0.1)")
	_ = viper.BindPFlag("secret-file", rootCmd.PersistentFlags().Lookup("secret-file"))
	_ = viper.BindEnv("secret-file", "DRIVE_SECRETS")
	_ = viper.BindPFlag("token-file", rootCmd.PersistentFlags().Lookup("token-file"))
	_ = viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	_ = viper.BindEnv("profile", "DRIVECTL_PROFILE")
	_ = viper.BindPFlag("mcp", rootCmd.PersistentFlags().Lookup("mcp"))
	_ = viper.BindPFlag("mcp-http", rootCmd.PersistentFlags().Lookup("mcp-http"))
}

func initConfig() {
	viper.AutomaticEnv()
}
