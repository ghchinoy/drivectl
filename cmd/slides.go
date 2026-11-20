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
	"strings"

	"github.com/spf13/cobra"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/drive"
)

var (
	slidesInspectSlide int
	slidesMatchCase    bool
	slidesNotesSlide   int
	slidesNotesSet     string
	slidesNotesFile    string
)

var slidesCmd = &cobra.Command{
	Use:     "slides",
	GroupID: GroupIntegration,
	Short:   "Interact with Google Slides",
	Long:    `A set of commands to inspect, create, and update Google Slides presentations.`,
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
			return printJSON(pres)
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
			return printJSON(pres)
		}

		editURL := fmt.Sprintf("https://docs.google.com/presentation/d/%s/edit", pres.PresentationId)
		ui.PrintSuccess("Created presentation %s %s %s", pres.Title, ui.ID("("+pres.PresentationId+")"), editURL)
		return nil
	},
}

var slidesInspectCmd = &cobra.Command{
	Use:     "inspect [presentationId]",
	Short:   "Lists slide IDs, element objectIds, text previews, and speaker notes IDs.",
	Long:    `Displays a compact structural outline of a Google Slides presentation for targeted editing with batchUpdate or slides subcommands.`,
	Example: `  drivectl slides inspect <presentation-id>
  drivectl slides inspect <presentation-id> --slide 3
  drivectl slides inspect <presentation-id> -O json`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		presentationID := args[0]

		pres, err := drive.FetchPresentation(slidesSvc, presentationID)
		if err != nil {
			return ui.ErrorWithHint(err, "Ensure the presentation ID is correct and the Google Slides API is enabled.")
		}

		outline := drive.SummarizePresentation(pres)
		if slidesInspectSlide > 0 {
			filtered, err := filterSlideSummary(outline.Slides, slidesInspectSlide)
			if err != nil {
				return ui.ErrorWithHint(err, "Specify a 1-based slide number within the presentation range.")
			}
			outline.Slides = filtered
		}

		if OutputFormat == "json" {
			return printJSON(outline)
		}

		printPresentationOutline(outline)
		return nil
	},
}

var slidesReplaceCmd = &cobra.Command{
	Use:     "replace [presentationId] [find] [replace]",
	Short:   "Replaces all occurrences of a text string across a presentation.",
	Long:    `Invokes the Slides API replaceAllText request to swap every occurrence of [find] with [replace] across all slides.`,
	Example: `  drivectl slides replace <presentation-id> "Speaker Name" "G. Hussain Chinoy"
  drivectl slides replace <presentation-id> "/ 14" "/ 21" --match-case -O json`,
	Args: cobra.ExactArgs(3),
	RunE: func(_ *cobra.Command, args []string) error {
		presentationID, findText, replaceText := args[0], args[1], args[2]

		changed, err := drive.ReplacePresentationText(slidesSvc, presentationID, findText, replaceText, slidesMatchCase)
		if err != nil {
			return ui.ErrorWithHint(err, "Ensure the presentation ID is correct and your OAuth token includes Slides write scope.")
		}

		if OutputFormat == "json" {
			return printJSON(map[string]any{
				"presentationId":     presentationID,
				"find":               findText,
				"replace":            replaceText,
				"occurrencesChanged": changed,
			})
		}

		ui.PrintSuccess("Replaced %d occurrence(s) of %q with %q", changed, findText, replaceText)
		return nil
	},
}

var slidesNotesCmd = &cobra.Command{
	Use:     "notes [presentationId]",
	Short:   "Reads or updates speaker notes on a specific slide.",
	Long:    `Reads the speaker notes on --slide N (1-based), or replaces them when --set or --file is provided.`,
	Example: `  drivectl slides notes <presentation-id> --slide 1
  drivectl slides notes <presentation-id> --slide 1 --set "Welcome everyone!"
  drivectl slides notes <presentation-id> --slide 2 --file ./slide2-notes.txt`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		presentationID := args[0]
		if slidesNotesSlide < 1 {
			return ui.ErrorWithHint(fmt.Errorf("--slide must be a positive 1-based slide number"), "Pass --slide 1 (or another slide index).")
		}

		setFlag := cmd.Flags().Changed("set")
		fileFlag := cmd.Flags().Changed("file")
		if setFlag && fileFlag {
			return ui.ErrorWithHint(fmt.Errorf("cannot specify both --set and --file"), "Use either --set or --file to update speaker notes.")
		}

		if setFlag || fileFlag {
			return runSlidesNotesUpdate(presentationID, setFlag)
		}
		return runSlidesNotesRead(presentationID)
	},
}

func runSlidesNotesRead(presentationID string) error {
	pres, err := drive.FetchPresentation(slidesSvc, presentationID)
	if err != nil {
		return ui.ErrorWithHint(err, "Ensure the presentation ID is correct and the Google Slides API is enabled.")
	}

	outline := drive.SummarizePresentation(pres)
	filtered, err := filterSlideSummary(outline.Slides, slidesNotesSlide)
	if err != nil {
		return ui.ErrorWithHint(err, "Specify a 1-based slide number within the presentation range.")
	}

	slide := filtered[0]
	if OutputFormat == "json" {
		return printJSON(map[string]any{
			"presentationId":       presentationID,
			"slide":                slide.Index,
			"objectId":             slide.ObjectID,
			"speakerNotesObjectId": slide.SpeakerNotesObjectID,
			"notes":                slide.SpeakerNotes,
		})
	}

	if slide.SpeakerNotes == "" {
		fmt.Println(ui.Muted("No speaker notes on this slide."))
		return nil
	}
	fmt.Println(slide.SpeakerNotes)
	return nil
}

func runSlidesNotesUpdate(presentationID string, useInline bool) error {
	notesText := slidesNotesSet
	if !useInline {
		raw, err := os.ReadFile(slidesNotesFile)
		if err != nil {
			wrapped := fmt.Errorf("unable to read notes file %q: %w", slidesNotesFile, err)
			return ui.ErrorWithHint(wrapped, "Ensure the path to the speaker notes file is correct.")
		}
		notesText = string(raw)
	}

	notesID, err := drive.UpdateSlideNotes(slidesSvc, presentationID, slidesNotesSlide, notesText)
	if err != nil {
		return ui.ErrorWithHint(err, "Ensure the slide index is valid and your OAuth token includes Slides write scope.")
	}

	if OutputFormat == "json" {
		return printJSON(map[string]any{
			"presentationId":       presentationID,
			"slide":                slidesNotesSlide,
			"speakerNotesObjectId": notesID,
			"notes":                notesText,
		})
	}

	ui.PrintSuccess("Updated speaker notes on slide %d %s", slidesNotesSlide, ui.ID("("+notesID+")"))
	return nil
}

func filterSlideSummary(slides []drive.SlideSummary, slideNum int) ([]drive.SlideSummary, error) {
	if slideNum < 1 || slideNum > len(slides) {
		return nil, fmt.Errorf("slide %d out of range (presentation has %d slides)", slideNum, len(slides))
	}
	return []drive.SlideSummary{slides[slideNum-1]}, nil
}

func printPresentationOutline(outline *drive.PresentationOutline) {
	if outline.Title != "" {
		fmt.Printf("%s %s\n", ui.Accent(outline.Title), ui.ID("("+outline.PresentationID+")"))
	}
	for _, s := range outline.Slides {
		notesTag := ""
		if s.SpeakerNotesObjectID != "" {
			notesTag = fmt.Sprintf(" notes=%s", s.SpeakerNotesObjectID)
		}
		fmt.Printf("Slide %d %s%s\n", s.Index, ui.ID("("+s.ObjectID+")"), ui.Muted(notesTag))
		for _, el := range s.Elements {
			preview := formatElementPreview(el.Text)
			if preview != "" {
				fmt.Printf("  %-24s %-6s %s\n", el.ObjectID, el.Kind, preview)
			} else {
				fmt.Printf("  %-24s %s\n", el.ObjectID, el.Kind)
			}
		}
	}
}

func formatElementPreview(text string) string {
	single := strings.Join(strings.Fields(text), " ")
	if len(single) > 80 {
		return single[:77] + "..."
	}
	return single
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func init() {
	rootCmd.AddCommand(slidesCmd)
	slidesCmd.AddCommand(slidesGetCmd)
	slidesCmd.AddCommand(slidesCreateCmd)
	slidesCmd.AddCommand(slidesInspectCmd)
	slidesCmd.AddCommand(slidesReplaceCmd)
	slidesCmd.AddCommand(slidesNotesCmd)

	slidesInspectCmd.Flags().IntVar(&slidesInspectSlide, "slide", 0, "Inspect only the specified 1-based slide number")
	slidesReplaceCmd.Flags().BoolVar(&slidesMatchCase, "match-case", false, "Match case when replacing text")
	slidesNotesCmd.Flags().IntVar(&slidesNotesSlide, "slide", 0, "1-based slide number (required)")
	slidesNotesCmd.Flags().StringVar(&slidesNotesSet, "set", "", "Replace speaker notes with the provided text")
	slidesNotesCmd.Flags().StringVar(&slidesNotesFile, "file", "", "Replace speaker notes with the contents of a local file")
}
