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

// CreatePresentation creates a new blank Google Slides presentation with the given title.
func CreatePresentation(slidesSvc *slides.Service, title string) (*slides.Presentation, error) {
	pres, err := slidesSvc.Presentations.Create(&slides.Presentation{
		Title: title,
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create presentation %q: %w", title, err)
	}
	return pres, nil
}

// BatchUpdatePresentation applies a batch of Google Slides API update requests to a presentation.
func BatchUpdatePresentation(slidesSvc *slides.Service, presentationID string, requests []*slides.Request) (*slides.BatchUpdatePresentationResponse, error) {
	resp, err := slidesSvc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to batch update presentation %q: %w", presentationID, err)
	}
	return resp, nil
}

// SlideElementSummary describes a page element on a Google Slides slide for inspection and editing.
type SlideElementSummary struct {
	ObjectID string `json:"objectId"`
	Kind     string `json:"kind"`
	Text     string `json:"text,omitempty"`
}

// SlideSummary describes a slide's 1-based index, object ID, elements, and speaker notes metadata.
type SlideSummary struct {
	Index                int                   `json:"index"`
	ObjectID             string                `json:"objectId"`
	SpeakerNotesObjectID string                `json:"speakerNotesObjectId,omitempty"`
	SpeakerNotes         string                `json:"speakerNotes,omitempty"`
	Elements             []SlideElementSummary `json:"elements"`
}

// PresentationOutline is a compact structural summary of a Google Slides presentation.
type PresentationOutline struct {
	PresentationID string         `json:"presentationId"`
	Title          string         `json:"title"`
	Slides         []SlideSummary `json:"slides"`
}

// SummarizePresentation builds a compact outline of slide IDs, element IDs, text previews,
// and speaker notes metadata from a *slides.Presentation.
func SummarizePresentation(pres *slides.Presentation) *PresentationOutline {
	if pres == nil {
		return &PresentationOutline{}
	}
	outline := &PresentationOutline{
		PresentationID: pres.PresentationId,
		Title:          pres.Title,
		Slides:         make([]SlideSummary, 0, len(pres.Slides)),
	}
	for idx, slide := range pres.Slides {
		if slide == nil {
			continue
		}
		outline.Slides = append(outline.Slides, summarizeSlide(idx+1, slide))
	}
	return outline
}

// ReplacePresentationText replaces all occurrences of find with replace across a presentation
// and returns the number of occurrences changed.
func ReplacePresentationText(slidesSvc *slides.Service, presentationID, find, replace string, matchCase bool) (int64, error) {
	req := &slides.Request{
		ReplaceAllText: &slides.ReplaceAllTextRequest{
			ContainsText: &slides.SubstringMatchCriteria{
				Text:      find,
				MatchCase: matchCase,
			},
			ReplaceText: replace,
		},
	}
	resp, err := BatchUpdatePresentation(slidesSvc, presentationID, []*slides.Request{req})
	if err != nil {
		return 0, err
	}
	var changed int64
	for _, reply := range resp.Replies {
		if reply != nil && reply.ReplaceAllText != nil {
			changed += reply.ReplaceAllText.OccurrencesChanged
		}
	}
	return changed, nil
}

// UpdateSlideNotes replaces the speaker notes text on the 1-based slideNum of a presentation
// and returns the speaker notes object ID that was updated.
func UpdateSlideNotes(slidesSvc *slides.Service, presentationID string, slideNum int, newNotes string) (string, error) {
	pres, err := FetchPresentation(slidesSvc, presentationID)
	if err != nil {
		return "", err
	}
	if slideNum < 1 || slideNum > len(pres.Slides) {
		return "", fmt.Errorf("slide %d out of range (presentation has %d slides)", slideNum, len(pres.Slides))
	}

	slide := pres.Slides[slideNum-1]
	notesID := speakerNotesObjectID(slide)
	if notesID == "" {
		return "", fmt.Errorf("slide %d does not have a speaker notes shape", slideNum)
	}

	var reqs []*slides.Request
	if extractSpeakerNotes(slide) != "" {
		reqs = append(reqs, &slides.Request{
			DeleteText: &slides.DeleteTextRequest{
				ObjectId:  notesID,
				TextRange: &slides.Range{Type: "ALL"},
			},
		})
	}
	if newNotes != "" {
		reqs = append(reqs, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId:       notesID,
				InsertionIndex: 0,
				Text:           newNotes,
			},
		})
	}
	if len(reqs) > 0 {
		if _, err := BatchUpdatePresentation(slidesSvc, presentationID, reqs); err != nil {
			return "", err
		}
	}
	return notesID, nil
}

func summarizeSlide(index int, slide *slides.Page) SlideSummary {
	summary := SlideSummary{
		Index:                index,
		ObjectID:             slide.ObjectId,
		SpeakerNotesObjectID: speakerNotesObjectID(slide),
		SpeakerNotes:         extractSpeakerNotes(slide),
		Elements:             make([]SlideElementSummary, 0, len(slide.PageElements)),
	}
	for _, elem := range slide.PageElements {
		if elem == nil {
			continue
		}
		summary.Elements = append(summary.Elements, summarizePageElement(elem))
	}
	return summary
}

func summarizePageElement(elem *slides.PageElement) SlideElementSummary {
	item := SlideElementSummary{ObjectID: elem.ObjectId, Kind: elementKind(elem)}
	if elem.Shape != nil && elem.Shape.Text != nil {
		item.Text = strings.TrimSpace(extractTextElements(elem.Shape.Text.TextElements))
	}
	return item
}

func elementKind(elem *slides.PageElement) string {
	switch {
	case elem.Shape != nil:
		return "shape"
	case elem.Image != nil:
		return "image"
	case elem.Table != nil:
		return "table"
	case elem.Line != nil:
		return "line"
	case elem.Video != nil:
		return "video"
	case elem.ElementGroup != nil:
		return "group"
	default:
		return "element"
	}
}

func speakerNotesObjectID(slide *slides.Page) string {
	if slide == nil || slide.SlideProperties == nil || slide.SlideProperties.NotesPage == nil {
		return ""
	}
	if slide.SlideProperties.NotesPage.NotesProperties == nil {
		return ""
	}
	return slide.SlideProperties.NotesPage.NotesProperties.SpeakerNotesObjectId
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
			b.WriteString(strings.ReplaceAll(te.TextRun.Content, "\x0b", "\n"))
		}
	}
	return b.String()
}
