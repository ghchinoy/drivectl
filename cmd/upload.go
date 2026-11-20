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

var (
	parentID       string
	uploadFileID   string
	uploadName     string
	uploadMimeType string
	uploadConvert  string
)

var uploadCmd = &cobra.Command{
	Use:     "upload [file]",
	GroupID: GroupCore,
	Short:   "Uploads a local file to Google Drive (or updates an existing file in place).",
	Long: `Uploads a file from your local filesystem to Google Drive.
Optionally specify --file-id to replace an existing file's content in place (preserving its
URL and permissions), a remote --name, a --parent folder ID, an explicit --mime-type, or
--convert (slides, docs, sheets) to convert the file into a native Google Workspace format.`,
	Example: `  drivectl upload path/to/my/file.txt
  drivectl upload ./report.pdf --name "Q3 Report.pdf" --parent <folder-id>
  drivectl upload ./deck.pptx --name "Workshop Deck" --convert slides
  drivectl upload ./deck.pptx --file-id <existing-id> --convert slides`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		filePath := args[0]

		targetMime, err := resolveUploadMimeType(uploadMimeType, uploadConvert)
		if err != nil {
			return ui.ErrorWithHint(err, "Specify either --convert (slides, docs, sheets) or --mime-type, not both.")
		}

		opts := drive.UploadOptions{
			FileID:   uploadFileID,
			Name:     uploadName,
			ParentID: parentID,
			MimeType: targetMime,
		}
		uploadedFile, err := drive.UploadFileWithOptions(driveSvc, filePath, opts)
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

		action := "uploaded"
		if uploadFileID != "" {
			action = "updated"
		}
		if uploadedFile.WebViewLink != "" {
			ui.PrintSuccess("Successfully %s file: %s (ID: %s) %s", action, uploadedFile.Name, uploadedFile.Id, uploadedFile.WebViewLink)
			return nil
		}
		ui.PrintSuccess("Successfully %s file: %s (ID: %s)", action, uploadedFile.Name, uploadedFile.Id)
		return nil
	},
}

func resolveUploadMimeType(explicitMime, convertTarget string) (string, error) {
	if explicitMime != "" && convertTarget != "" {
		return "", fmt.Errorf("cannot specify both --mime-type and --convert")
	}
	if convertTarget != "" {
		return drive.ResolveConvertMimeType(convertTarget)
	}
	return explicitMime, nil
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringVarP(&parentID, "parent", "p", "", "Parent folder ID to upload to")
	uploadCmd.Flags().StringVar(&uploadFileID, "file-id", "", "Existing Drive file ID to update in place (preserves share URL and permissions)")
	uploadCmd.Flags().StringVarP(&uploadName, "name", "n", "", "Remote file name in Google Drive (defaults to local filename)")
	uploadCmd.Flags().StringVar(&uploadMimeType, "mime-type", "", "Target MIME type in Google Drive metadata")
	uploadCmd.Flags().StringVar(&uploadConvert, "convert", "", "Convert file on upload to a native Google Workspace format (slides, docs, sheets)")
}
