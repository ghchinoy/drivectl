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
	"strings"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/drive"
	"github.com/spf13/cobra"
)

var revisionsCmd = &cobra.Command{
	Use:     "revisions <file-id>",
	GroupID: GroupCore,
	Short:   "Lists revisions for a Google Drive file or Google Doc",
	Long:    `Retrieves and displays the revision history for a given file ID from the Google Drive API.`,
	Example: `  drivectl revisions <file-id>`,
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		fileID := args[0]
		revisions, err := drive.ListRevisions(driveSvc, fileID)
		if err != nil {
			hint := revisionsErrorHint(fileID, err)
			wrapped := fmt.Errorf("unable to retrieve revisions: %w", err)
			return ui.ErrorWithHint(wrapped, hint)
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(revisions, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		if len(revisions) == 0 {
			fmt.Println(ui.Muted("No revisions found for this file."))
			return nil
		}

		fmt.Println(ui.Accent(fmt.Sprintf("Revisions for %s:", fileID)))
		for _, rev := range revisions {
			author := "Unknown"
			if rev.LastModifyingUser != nil {
				author = rev.LastModifyingUser.DisplayName
			}
			fmt.Printf("- %s | %s by %s\n", ui.ID(rev.Id), ui.Muted(rev.ModifiedTime), author)
		}

		return nil
	},
}

func revisionsErrorHint(fileID string, err error) string {
	hint := "Check if the file ID is correct and you have permission to view revisions."
	if !strings.Contains(err.Error(), "500") && !strings.Contains(err.Error(), "403") {
		return hint
	}
	file, describeErr := drive.DescribeFile(driveSvc, fileID)
	if describeErr != nil {
		return hint
	}
	if file.CopyRequiresWriterPermission {
		return "This document has download/copy restrictions enabled, which can cause the Google API to reject revisions requests."
	}
	return "The Google API returned an error. This can sometimes happen if the document has DLP policies or other strict restrictions."
}

func init() {
	rootCmd.AddCommand(revisionsCmd)
}
