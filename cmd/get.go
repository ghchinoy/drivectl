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
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/api/docs/v1"
)

var (
	outputFile string
	format     string
	tabIndex   int
)

// renderBodyAsText converts a Google Docs Body object to a plain text string.
func renderBodyAsText(body *docs.Body) string {
	var text strings.Builder
	if body == nil || body.Content == nil {
		return ""
	}
	for _, element := range body.Content {
		if element.Paragraph != nil {
			for _, pElem := range element.Paragraph.Elements {
				if pElem.TextRun != nil {
					text.WriteString(pElem.TextRun.Content)
				}
			}
		}
	}
	return text.String()
}

var formatMap = map[string]string{
	"pdf":      "application/pdf",
	"docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"html":     "text/html",
	"zip":      "application/zip",
	"epub":     "application/epub+zip",
	"txt":      "text/plain",
	"md":       "text/markdown",
	"markdown": "text/markdown",
}

var getCmd = &cobra.Command{
	Use:   "get [fileId]",
	Short: "Downloads a file from Google Drive. Google Docs can be exported to different formats.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileId := args[0]

		// If a tab index is specified, we must use the Docs API.
		if tabIndex >= 0 {
			doc, err := docsSvc.Documents.Get(fileId).IncludeTabsContent(true).Do()
			if err != nil {
				return fmt.Errorf("unable to retrieve document with tabs: %w", err)
			}

			if tabIndex >= len(doc.Tabs) {
				return fmt.Errorf("invalid tab index: %d. Document only has %d tabs", tabIndex, len(doc.Tabs))
			}

			theTab := doc.Tabs[tabIndex]
			textContent := renderBodyAsText(theTab.DocumentTab.Body)
			content := []byte(textContent)

			if outputFile != "" {
				err := os.WriteFile(outputFile, content, 0644)
				if err != nil {
					return fmt.Errorf("failed to write to output file %s: %w", outputFile, err)
				}
				fmt.Printf("Successfully saved tab %d to %s\n", tabIndex, outputFile)
			} else {
				fmt.Println(string(content))
			}
			return nil
		}

		// Default behavior: use the Drive API for direct export or download.
		file, err := driveSvc.Files.Get(fileId).Fields("mimeType", "name").Do()
		if err != nil {
			return fmt.Errorf("unable to retrieve file metadata: %w", err)
		}

		var content []byte

		if file.MimeType == "application/vnd.google-apps.document" {
			exportMimeType, ok := formatMap[strings.ToLower(format)]
			if !ok && format != "" {
				return fmt.Errorf("invalid format for Google Doc: %s. Valid formats are: pdf, docx, html, zip, epub, txt, md", format)
			}
			if exportMimeType == "" {
				exportMimeType = "text/plain" // Default to plain text
			}

			resp, err := driveSvc.Files.Export(fileId, exportMimeType).Download()
			if err != nil {
				return fmt.Errorf("unable to export Google Doc: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("unable to read exported content: %w", err)
			}
			content = body
		} else {
			// It's a regular file, download it directly
			resp, err := driveSvc.Files.Get(fileId).Download()
			if err != nil {
				return fmt.Errorf("unable to download file: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("unable to read file content: %w", err)
			}
			content = body
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
			if format != "" && format != "txt" && format != "html" && format != "md" {
				fmt.Printf("Successfully downloaded file content. Use the -o flag to save it to a file.\n")
			} else {
				fmt.Println(string(content))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path to save the output file")
	getCmd.Flags().StringVar(&format, "format", "", "Export format for Google Docs (e.g., pdf, docx, html, txt, md)")
	getCmd.Flags().IntVar(&tabIndex, "tab-index", -1, "Index of the tab to get content from")
}
