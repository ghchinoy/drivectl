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
	"io/ioutil"
	"strings"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/spf13/cobra"
)

var (
	addTabTitle        string
	addTabMarkdownFile string
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Interact with Google Docs",
	Long:  `A set of commands to interact with Google Docs.`,
}

var docsTabsCmd = &cobra.Command{
	Use:   "tabs [documentId]",
	Short: "Lists the tabs within a Google Doc.",
	Long:  `Lists the available tabs for a given Google Doc by their index number. This uses the Google Docs API.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		documentId := args[0]

		tabs, err := drive.GetTabs(docsSvc, documentId)
		if err != nil {
			return err
		}

		fmt.Println("Tabs:")
		if len(tabs) == 0 {
			fmt.Println("No tabs found.")
		} else {
			var printTabs func(tabs []*drive.TabInfo, level int)
			printTabs = func(tabs []*drive.TabInfo, level int) {
				for _, tab := range tabs {
					fmt.Printf("%s%s (%s)\n", strings.Repeat("\t", level), tab.Title, tab.TabID)
					if len(tab.Children) > 0 {
						printTabs(tab.Children, level+1)
					}
				}
			}
			printTabs(tabs, 0)
		}
		return nil
	},
}

var docsCreateCmd = &cobra.Command{
	Use:   "create [title] [markdown-file]",
	Short: "Creates a new Google Doc from a Markdown file.",
	Long:  `Creates a new Google Doc from a Markdown file.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]
		markdownFile := args[1]

		content, err := ioutil.ReadFile(markdownFile)
		if err != nil {
			return fmt.Errorf("unable to read markdown file: %w", err)
		}

		doc, err := drive.CreateDocFromMarkdown(docsSvc, title, string(content))
		if err != nil {
			return err
		}

		fmt.Printf("Successfully created document %s (%s)\n", doc.Title, doc.DocumentId)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.AddCommand(docsTabsCmd)
	docsCmd.AddCommand(docsCreateCmd)
}