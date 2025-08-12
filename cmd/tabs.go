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

	"github.com/spf13/cobra"
)

var tabsCmd = &cobra.Command{
	Use:   "tabs [documentId]",
	Short: "Lists the tabs within a Google Doc.",
	Long:  `Lists the available tabs for a given Google Doc by their index number. This uses the Google Docs API.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		documentId := args[0]

		doc, err := docsSvc.Documents.Get(documentId).IncludeTabsContent(true).Do()
		if err != nil {
			return fmt.Errorf("unable to retrieve document with tabs: %w", err)
		}

		fmt.Println("Tabs:")
		if len(doc.Tabs) == 0 {
			fmt.Println("No tabs found.")
		} else {
			for i := range doc.Tabs {
				fmt.Printf("Tab %d\n", i)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tabsCmd)
}
