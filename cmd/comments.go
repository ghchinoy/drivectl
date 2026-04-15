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

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/spf13/cobra"
)

var commentsCmd = &cobra.Command{
	Use:     "comments <file-id>",
	GroupID: GroupCore,
	Short:   "Lists comments for a Google Drive file or Google Doc",
	Long:    `Retrieves and displays the comment history for a given file ID from the Google Drive API.`,
	Example: `  drivectl comments <file-id>`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileId := args[0]
		comments, err := drive.ListComments(driveSvc, fileId)
		if err != nil {
			hint := "Check if the file ID is correct and you have permission to view comments."
			if strings.Contains(err.Error(), "500") || strings.Contains(err.Error(), "403") {
				if file, describeErr := drive.DescribeFile(driveSvc, fileId); describeErr == nil {
					if file.CopyRequiresWriterPermission {
						hint = "This document has download/copy restrictions enabled, which can cause the Google API to reject comments requests."
					} else {
						hint = "The Google API returned an error. This can sometimes happen if the document has DLP policies or other strict restrictions."
					}
				}
			}
			return ui.ErrorWithHint(fmt.Errorf("unable to retrieve comments: %w", err), hint)
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(comments, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		if len(comments) == 0 {
			fmt.Println(ui.Muted("No comments found for this file."))
			return nil
		}

		fmt.Println(ui.Accent(fmt.Sprintf("Comments for %s:", fileId)))
		for _, comment := range comments {
			author := "Unknown"
			if comment.Author != nil {
				author = comment.Author.DisplayName
			}
			
			// Display the main comment
			fmt.Printf("\n[%s] %s\n", ui.Muted(comment.CreatedTime), ui.ID(author))
			if comment.QuotedFileContent != nil && comment.QuotedFileContent.Value != "" {
				fmt.Printf("%s\n", ui.Muted(fmt.Sprintf("  > %s", comment.QuotedFileContent.Value)))
			}
			fmt.Printf("  %s\n", comment.Content)
			
			// Display replies
			for _, reply := range comment.Replies {
				replyAuthor := "Unknown"
				if reply.Author != nil {
					replyAuthor = reply.Author.DisplayName
				}
				fmt.Printf("    - [%s] %s: %s\n", ui.Muted(reply.CreatedTime), ui.ID(replyAuthor), reply.Content)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commentsCmd)
}
