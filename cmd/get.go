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
	"os"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	outputFile string
	format     string
	tabID      string
)

var getCmd = &cobra.Command{
	Use:     "get [fileId]",
	GroupID: GroupCore,
	Short:   "Downloads a file or exports a Google Doc.",
	Long: `Downloads a file from Google Drive.
For standard files (PDFs, images, etc.), it downloads the raw content.
For Google Docs, it can export the entire document to various formats (txt, md, pdf, etc.) using the --format flag.
It can also extract the plain text content of a single tab from a Google Doc using the --tab-id flag.`,
	Example: `  drivectl get <file-id>
  drivectl get <google-doc-id> --format md -o my-doc.md
  drivectl get <google-doc-id> --tab-id <tab-id>`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		fileID := args[0]

		content, err := drive.FetchFile(driveSvc, docsSvc, fileID, format, tabID)
		if err != nil {
			return err
		}

		if OutputFormat == "json" {
			res := map[string]any{
				"fileId":  fileID,
				"content": string(content),
				"format":  format,
				"tabId":   tabID,
			}
			b, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, content, 0o600); err != nil {
				wrapped := fmt.Errorf("failed to write to output file %q: %w", outputFile, err)
				return ui.ErrorWithHint(wrapped, "Check file permissions and path.")
			}
			ui.PrintSuccess("Saved file to %s", outputFile)
			return nil
		}

		if format != "" && format != "txt" && format != "html" && format != "md" {
			ui.PrintSuccess("Downloaded file content. Use the -o flag to save it to a file.")
			return nil
		}

		fmt.Println(string(content))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path to save the output file")
	getCmd.Flags().StringVar(&format, "format", "", "Export format for Google Docs (e.g., pdf, docx, html, txt, md)")
	getCmd.Flags().StringVar(&tabID, "tab-id", "", "ID of the tab to get content from")
}
