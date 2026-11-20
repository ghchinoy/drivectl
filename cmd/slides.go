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

var slidesCmd = &cobra.Command{
	Use:     "slides",
	GroupID: GroupIntegration,
	Short:   "Interact with Google Slides",
	Long:    `A set of commands to inspect and create Google Slides presentations.`,
}

var slidesGetCmd = &cobra.Command{
	Use:     "get [presentationId]",
	Short:   "Renders a Google Slides presentation as Markdown (or JSON with -O json).",
	Long:    `Retrieves a Google Slides presentation via the Slides API and renders slide text, tables, and speaker notes as Markdown.`,
	Example: `  drivectl slides get <presentation-id>
  drivectl slides get <presentation-id> -O json`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		presentationID := args[0]

		pres, err := drive.FetchPresentation(slidesSvc, presentationID)
		if err != nil {
			return ui.ErrorWithHint(err, "Ensure the presentation ID is correct and the Google Slides API is enabled.")
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(pres, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		fmt.Print(drive.RenderPresentationAsMarkdown(pres))
		return nil
	},
}

var slidesCreateCmd = &cobra.Command{
	Use:     "create [title]",
	Short:   "Creates a new Google Slides presentation.",
	Long:    `Creates a new blank Google Slides presentation with the specified title using the Google Slides API.`,
	Example: `  drivectl slides create "DevFest 2026 Workshop"
  drivectl slides create "DevFest 2026 Workshop" -O json`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		title := args[0]

		pres, err := drive.CreatePresentation(slidesSvc, title)
		if err != nil {
			return ui.ErrorWithHint(err, "Ensure the Google Slides API is enabled and re-run 'drivectl auth login' if your cached token lacks Slides write scope.")
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(pres, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		editURL := fmt.Sprintf("https://docs.google.com/presentation/d/%s/edit", pres.PresentationId)
		ui.PrintSuccess("Created presentation %s %s %s", pres.Title, ui.ID("("+pres.PresentationId+")"), editURL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(slidesCmd)
	slidesCmd.AddCommand(slidesGetCmd)
	slidesCmd.AddCommand(slidesCreateCmd)
}
