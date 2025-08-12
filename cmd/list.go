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
	"fmt"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/spf13/cobra"
)

var (
	query string
	limit int64
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists files and folders in Google Drive.",
	Long: `Lists files and folders in your Google Drive.
Supports powerful filtering using the Google Drive query language via the --query flag.
For more information on query syntax, see: https://developers.google.com/drive/api/v3/search-files`,
	RunE: func(cmd *cobra.Command, args []string) error {
		files, err := drive.ListFiles(driveSvc, limit, query)
		if err != nil {
			return fmt.Errorf("unable to retrieve files: %w", err)
		}

		fmt.Println("Files:")
		if len(files) == 0 {
			fmt.Println("No files found.")
		} else {
			for _, i := range files {
				fmt.Printf("%s (%s)\n", i.Name, i.Id)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&query, "query", "q", "", "Query to filter files")
	listCmd.Flags().Int64Var(&limit, "limit", 100, "Maximum number of files to return")
}
