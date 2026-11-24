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

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ghchinoy/drivectl/pkg/drive"
)

func boolPtr(v bool) *bool { return &v }

func readOnlyAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    true,
		DestructiveHint: boolPtr(false),
		IdempotentHint:  true,
		OpenWorldHint:   boolPtr(true),
	}
}

func additiveAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(false),
		IdempotentHint:  false,
		OpenWorldHint:   boolPtr(true),
	}
}

func mutatingAnnotations(title string, idempotent bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(true),
		IdempotentHint:  idempotent,
		OpenWorldHint:   boolPtr(true),
	}
}

func discoveryToolAnnotations(endpoint, httpMethod string) *mcp.ToolAnnotations {
	if strings.EqualFold(httpMethod, http.MethodGet) {
		return readOnlyAnnotations(endpoint)
	}
	if strings.HasSuffix(endpoint, ".create") {
		return additiveAnnotations(endpoint)
	}
	return mutatingAnnotations(endpoint, strings.EqualFold(httpMethod, http.MethodPut))
}

type docsCreateFromMarkdownArgs struct {
	Title    string `json:"title" jsonschema:"Title for the new Google Doc"`
	Markdown string `json:"markdown" jsonschema:"Markdown content to convert into Google Docs AST formatting"`
}

type docsExportMarkdownArgs struct {
	DocumentID string `json:"documentId" jsonschema:"Google Doc ID to export"`
	TabID      string `json:"tabId,omitempty" jsonschema:"Optional tab ID to export instead of the full document"`
}

type docsTabsArgs struct {
	DocumentID string `json:"documentId" jsonschema:"Google Doc ID whose tab hierarchy should be inspected"`
}

type sheetsExportCSVArgs struct {
	SpreadsheetID string `json:"spreadsheetId" jsonschema:"Google Spreadsheet ID"`
	SheetName     string `json:"sheetName" jsonschema:"Name of the sheet tab to export as RFC 4180 CSV"`
}

type slidesGetMarkdownArgs struct {
	PresentationID string `json:"presentationId" jsonschema:"Google Slides presentation ID"`
}

type slidesInspectArgs struct {
	PresentationID string `json:"presentationId" jsonschema:"Google Slides presentation ID"`
	Slide          int    `json:"slide,omitempty" jsonschema:"Optional 1-based slide number to inspect"`
}

type slidesUpdateNotesArgs struct {
	PresentationID string `json:"presentationId" jsonschema:"Google Slides presentation ID"`
	Slide          int    `json:"slide" jsonschema:"1-based slide index whose speaker notes will be replaced"`
	Notes          string `json:"notes" jsonschema:"New speaker notes text"`
}

type slidesReplaceTextArgs struct {
	PresentationID string `json:"presentationId" jsonschema:"Google Slides presentation ID"`
	Find           string `json:"find" jsonschema:"Exact substring to search for across all slides"`
	Replace        string `json:"replace" jsonschema:"Replacement text"`
	MatchCase      bool   `json:"matchCase,omitempty" jsonschema:"Whether text matching should be case-sensitive"`
}

func registerNativeTools(server *mcp.Server, clientProvider ClientProvider) {
	registerDocsNativeTools(server, clientProvider)
	registerSheetsNativeTools(server, clientProvider)
	registerSlidesNativeTools(server, clientProvider)
}

func registerDocsNativeTools(server *mcp.Server, clientProvider ClientProvider) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "drivectl_docs_create_from_markdown",
		Description: "Create a formatted Google Doc from Markdown using drivectl's Goldmark-to-Docs AST converter.",
		Annotations: additiveAnnotations("Create Google Doc from Markdown"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args docsCreateFromMarkdownArgs) (*mcp.CallToolResult, any, error) {
		svcs, err := servicesForProvider(ctx, clientProvider)
		if err != nil {
			return nil, nil, err
		}
		doc, err := drive.CreateDocFromMarkdown(svcs.Docs, args.Title, args.Markdown)
		if err != nil {
			return nil, nil, err
		}
		return jsonTextToolResult(map[string]any{
			"documentId":  doc.DocumentId,
			"title":       doc.Title,
			"webViewLink": fmt.Sprintf("https://docs.google.com/document/d/%s/edit", doc.DocumentId),
		})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "drivectl_docs_export_markdown",
		Description: "Export a Google Doc (or a specific tab) as clean Markdown.",
		Annotations: readOnlyAnnotations("Export Google Doc as Markdown"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args docsExportMarkdownArgs) (*mcp.CallToolResult, any, error) {
		svcs, err := servicesForProvider(ctx, clientProvider)
		if err != nil {
			return nil, nil, err
		}
		data, err := drive.FetchFile(svcs.Drive, svcs.Docs, args.DocumentID, "md", args.TabID)
		if err != nil {
			return nil, nil, err
		}
		return textToolResult(string(data)), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "drivectl_docs_tabs",
		Description: "List the hierarchical tab and child-tab tree for a Google Doc.",
		Annotations: readOnlyAnnotations("List Google Doc Tabs"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args docsTabsArgs) (*mcp.CallToolResult, any, error) {
		svcs, err := servicesForProvider(ctx, clientProvider)
		if err != nil {
			return nil, nil, err
		}
		tabs, err := drive.FetchTabs(svcs.Docs, args.DocumentID)
		if err != nil {
			return nil, nil, err
		}
		return jsonTextToolResult(tabs)
	})
}

func registerSheetsNativeTools(server *mcp.Server, clientProvider ClientProvider) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "drivectl_sheets_export_csv",
		Description: "Export a Google Sheets tab as RFC 4180-escaped CSV.",
		Annotations: readOnlyAnnotations("Export Google Sheet Tab as CSV"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args sheetsExportCSVArgs) (*mcp.CallToolResult, any, error) {
		svcs, err := servicesForProvider(ctx, clientProvider)
		if err != nil {
			return nil, nil, err
		}
		csvData, err := drive.FetchSheetAsCSV(svcs.Sheets, args.SpreadsheetID, args.SheetName)
		if err != nil {
			return nil, nil, err
		}
		return textToolResult(csvData), nil, nil
	})
}

func registerSlidesNativeTools(server *mcp.Server, clientProvider ClientProvider) {
	registerSlidesReadTools(server, clientProvider)
	registerSlidesWriteTools(server, clientProvider)
}

func registerSlidesReadTools(server *mcp.Server, clientProvider ClientProvider) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "drivectl_slides_get_markdown",
		Description: "Render a Google Slides presentation (slide text, tables, and speaker notes) as structured Markdown.",
		Annotations: readOnlyAnnotations("Render Google Slides as Markdown"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args slidesGetMarkdownArgs) (*mcp.CallToolResult, any, error) {
		svcs, err := servicesForProvider(ctx, clientProvider)
		if err != nil {
			return nil, nil, err
		}
		md, err := drive.FetchPresentationAsMarkdown(svcs.Slides, args.PresentationID)
		if err != nil {
			return nil, nil, err
		}
		return textToolResult(md), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "drivectl_slides_inspect",
		Description: "Inspect slide objectIds, page element objectIds, text previews, and speakerNotesObjectIds for a Google Slides presentation.",
		Annotations: readOnlyAnnotations("Inspect Google Slides Structure"),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args slidesInspectArgs) (*mcp.CallToolResult, any, error) {
		svcs, err := servicesForProvider(ctx, clientProvider)
		if err != nil {
			return nil, nil, err
		}
		pres, err := drive.FetchPresentation(svcs.Slides, args.PresentationID)
		if err != nil {
			return nil, nil, err
		}
		outline := drive.SummarizePresentation(pres)
		if args.Slide > 0 {
			if args.Slide > len(outline.Slides) {
				return nil, nil, fmt.Errorf("slide %d out of range (presentation has %d slides)", args.Slide, len(outline.Slides))
			}
			outline.Slides = []drive.SlideSummary{outline.Slides[args.Slide-1]}
		}
		return jsonTextToolResult(outline)
	})
}

func registerSlidesWriteTools(server *mcp.Server, clientProvider ClientProvider) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "drivectl_slides_update_notes",
		Description: "Replace speaker notes on a 1-based slide index in a Google Slides presentation.",
		Annotations: mutatingAnnotations("Update Google Slides Speaker Notes", true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args slidesUpdateNotesArgs) (*mcp.CallToolResult, any, error) {
		if args.Slide < 1 {
			return nil, nil, errors.New("slide must be a positive 1-based slide number")
		}
		svcs, err := servicesForProvider(ctx, clientProvider)
		if err != nil {
			return nil, nil, err
		}
		notesID, err := drive.UpdateSlideNotes(svcs.Slides, args.PresentationID, args.Slide, args.Notes)
		if err != nil {
			return nil, nil, err
		}
		return jsonTextToolResult(map[string]any{
			"presentationId":       args.PresentationID,
			"slide":                args.Slide,
			"speakerNotesObjectId": notesID,
			"notes":                args.Notes,
		})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "drivectl_slides_replace_text",
		Description: "Replace all occurrences of a text string across a Google Slides presentation.",
		Annotations: mutatingAnnotations("Replace Text in Google Slides", true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args slidesReplaceTextArgs) (*mcp.CallToolResult, any, error) {
		svcs, err := servicesForProvider(ctx, clientProvider)
		if err != nil {
			return nil, nil, err
		}
		changed, err := drive.ReplacePresentationText(svcs.Slides, args.PresentationID, args.Find, args.Replace, args.MatchCase)
		if err != nil {
			return nil, nil, err
		}
		return jsonTextToolResult(map[string]any{
			"presentationId":     args.PresentationID,
			"find":               args.Find,
			"replace":            args.Replace,
			"occurrencesChanged": changed,
		})
	})
}

func servicesForProvider(ctx context.Context, clientProvider ClientProvider) (*drive.Services, error) {
	httpClient, err := clientProvider(ctx)
	if err != nil {
		return nil, err
	}
	return drive.NewServicesFromClient(ctx, httpClient)
}

func textToolResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func jsonTextToolResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("encoding tool result: %w", err)
	}
	return textToolResult(string(b)), nil, nil
}
