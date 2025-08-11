
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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	noBrowserAuth bool
	driveSvc      *drive.Service
	docsSvc       *docs.Service
)

var rootCmd = &cobra.Command{
	Use:   "drivectl",
	Short: "A CLI for Google Drive and Docs.",
	Long: `drivectl is a powerful command-line tool for interacting with your Google Drive files.
It allows you to list, describe, and download files, with advanced support for
Google Docs, including exporting to multiple formats and accessing individual tabs.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		secretFile := viper.GetString("secret-file")
		if secretFile == "" {
			return fmt.Errorf("client secret file not set. Please use the --secret-file flag or set the DRIVE_SECRETS environment variable")
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
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("secret-file", "", "path to your client secrets file")
	rootCmd.PersistentFlags().BoolVar(&noBrowserAuth, "no-browser-auth", false, "do not open a browser for authentication")
	viper.BindPFlag("secret-file", rootCmd.PersistentFlags().Lookup("secret-file"))
	viper.BindEnv("secret-file", "DRIVE_SECRETS")
}

func initConfig() {
	viper.AutomaticEnv()
}
