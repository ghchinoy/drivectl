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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/drive"
)

var parentID string

var uploadCmd = &cobra.Command{
	Use:     "upload [file]",
	GroupID: GroupCore,
	Short:   "Uploads a local file to Google Drive.",
	Long: `Uploads a file from your local filesystem to Google Drive.
Optionally, specify a parent folder ID to upload the file to a specific folder.`,
	Example: `  drivectl upload path/to/my/file.txt
  drivectl upload path/to/my/file.txt --parent <folder-id>`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		filePath := args[0]

		uploadedFile, err := drive.UploadFile(driveSvc, filePath, parentID)
		if err != nil {
			return ui.ErrorWithHint(err, "Ensure the file path is correct and you have permission to upload.")
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(uploadedFile, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		ui.PrintSuccess("Successfully uploaded file: %s (ID: %s)", uploadedFile.Name, uploadedFile.Id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringVarP(&parentID, "parent", "p", "", "Parent folder ID to upload to")
}
