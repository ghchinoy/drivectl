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
	"os"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/spf13/cobra"
)

var (
	getFormat    string
	getTabId     string
	noImages     bool
	outputFile   string
)

var getCmd = &cobra.Command{
	Use:   "get [fileId]",
	Short: "Downloads a file or exports a Google Doc.",
	Long: `Downloads a file from Google Drive.
For standard files (PDFs, images, etc.), it downloads the raw content.
For Google Docs, it can export the entire document to various formats (txt, md, pdf, etc.) using the --format flag.
It can also extract the plain text content of a single tab from a Google Doc using the --tab-id flag.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileId := args[0]

		content, err := drive.GetFile(driveSvc, docsSvc, fileId, getFormat, getTabId, noImages)
		if err != nil {
			return err
		}

		if outputFile != "" {
			err := os.WriteFile(outputFile, content, 0644)
			if err != nil {
				return fmt.Errorf("failed to write to output file %s: %w", outputFile, err)
			}
			fmt.Printf("Successfully saved file to %s\n", outputFile)
		} else {
			// For binary formats like pdf, docx, etc., printing to console is not useful.
			// We will just print a success message instead.
			if getFormat != "" && getFormat != "txt" && getFormat != "html" && getFormat != "md" {
				fmt.Printf("Successfully downloaded file content. Use the -o flag to save it to a file.\n")
			} else {
				fmt.Println(string(content))
			}
		}

		return nil
	},
}

func init() {
	getCmd.Flags().StringVar(&getFormat, "format", "", "The format to export the file in. If not provided, the file will be downloaded in its native format.")
	getCmd.Flags().StringVar(&getTabId, "tab-id", "", "The ID of the tab to get. If provided, only the content of this tab will be returned.")
	getCmd.Flags().BoolVar(&noImages, "no-images", false, "Exclude images from the document content.")
	getCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path to save the output file")
	rootCmd.AddCommand(getCmd)
}
