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
	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	query string
	limit int64
)

var listCmd = &cobra.Command{
	Use:     "list",
	GroupID: GroupCore,
	Short:   "Lists files and folders in Google Drive.",
	Long: `Lists files and folders in your Google Drive.
Supports powerful filtering using the Google Drive query language via the --query flag.
For more information on query syntax, see: https://developers.google.com/drive/api/v3/search-files`,
	Example: `  drivectl list
  drivectl list --limit 20
  drivectl list -q "mimeType='application/vnd.google-apps.document'"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		files, err := drive.ListFiles(driveSvc, limit, query)
		if err != nil {
			return ui.ErrorWithHint(fmt.Errorf("unable to retrieve files: %w", err), "Check your query syntax and ensure you have network access.")
		}

		fmt.Println(ui.Accent("Files:"))
		if len(files) == 0 {
			fmt.Println(ui.Muted("No files found."))
		} else {
			for _, i := range files {
				fmt.Printf("%s %s\n", i.Name, ui.ID("("+i.Id+")"))
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
