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

package drive

import (
	"fmt"
	"strings"

	"google.golang.org/api/slides/v1"
)

// FetchPresentation retrieves a Google Slides presentation by ID.
func FetchPresentation(slidesSvc *slides.Service, presentationID string) (*slides.Presentation, error) {
	pres, err := slidesSvc.Presentations.Get(presentationID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve presentation: %w", err)
	}
	return pres, nil
}

// FetchPresentationAsMarkdown retrieves a Google Slides presentation and renders its slide text,
// tables, and speaker notes as structured Markdown.
func FetchPresentationAsMarkdown(slidesSvc *slides.Service, presentationID string) (string, error) {
	pres, err := FetchPresentation(slidesSvc, presentationID)
	if err != nil {
		return "", err
	}
	return RenderPresentationAsMarkdown(pres), nil
}

// RenderPresentationAsMarkdown converts a *slides.Presentation AST (slides, shapes, tables,
// and speaker notes) into structured Markdown.
func RenderPresentationAsMarkdown(pres *slides.Presentation) string {
	if pres == nil {
		return ""
	}
	var out strings.Builder
	if pres.Title != "" {
		fmt.Fprintf(&out, "# %s\n\n", pres.Title)
	}
	for idx, slide := range pres.Slides {
		if slide == nil {
			continue
		}
		renderSlide(&out, idx+1, slide)
	}
	return out.String()
}

func renderSlide(out *strings.Builder, slideNum int, slide *slides.Page) {
	fmt.Fprintf(out, "## Slide %d\n\n", slideNum)
	for _, elem := range slide.PageElements {
		renderPageElement(out, elem)
	}
	if notes := extractSpeakerNotes(slide); notes != "" {
		fmt.Fprintf(out, "### Speaker Notes\n\n%s\n", notes)
	}
}

func renderPageElement(out *strings.Builder, elem *slides.PageElement) {
	if elem == nil {
		return
	}
	if elem.Shape != nil && elem.Shape.Text != nil {
		if text := extractTextElements(elem.Shape.Text.TextElements); text != "" {
			out.WriteString(text)
			if !strings.HasSuffix(text, "\n") {
				out.WriteString("\n")
			}
		}
	}
	if elem.Table != nil {
		renderTable(out, elem.Table)
	}
}

func renderTable(out *strings.Builder, table *slides.Table) {
	for _, row := range table.TableRows {
		if row == nil {
			continue
		}
		cells := make([]string, 0, len(row.TableCells))
		for _, cell := range row.TableCells {
			if cell == nil || cell.Text == nil {
				cells = append(cells, "")
				continue
			}
			cellText := strings.TrimSpace(extractTextElements(cell.Text.TextElements))
			cells = append(cells, strings.ReplaceAll(cellText, "\n", " "))
		}
		fmt.Fprintf(out, "| %s |\n", strings.Join(cells, " | "))
	}
	out.WriteString("\n")
}

func extractSpeakerNotes(slide *slides.Page) string {
	if slide.SlideProperties == nil || slide.SlideProperties.NotesPage == nil {
		return ""
	}
	notesPage := slide.SlideProperties.NotesPage
	notesID := ""
	if notesPage.NotesProperties != nil {
		notesID = notesPage.NotesProperties.SpeakerNotesObjectId
	}
	var notes strings.Builder
	for _, elem := range notesPage.PageElements {
		if elem == nil || elem.Shape == nil || elem.Shape.Text == nil {
			continue
		}
		if notesID != "" && elem.ObjectId != notesID {
			continue
		}
		notes.WriteString(extractTextElements(elem.Shape.Text.TextElements))
	}
	return strings.TrimSpace(notes.String())
}

func extractTextElements(elements []*slides.TextElement) string {
	var b strings.Builder
	for _, te := range elements {
		if te != nil && te.TextRun != nil {
			b.WriteString(te.TextRun.Content)
		}
	}
	return b.String()
}
