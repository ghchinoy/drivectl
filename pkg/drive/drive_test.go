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
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"
)

func TestMarkdownToDocsRequests(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		wantLen  int
	}{
		{
			name:     "heading_and_paragraph",
			markdown: "# Title\n\nHello *italic* and **bold** with [link](https://example.com).",
			wantLen:  13,
		},
		{
			name:     "unordered_and_ordered_lists",
			markdown: "- First *item*\n- Second **item**\n\n1. One\n2. Two",
			wantLen:  10,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MarkdownToDocsRequests(tc.markdown)
			if err != nil {
				t.Fatalf("MarkdownToDocsRequests(%q) error = %v", tc.markdown, err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len(MarkdownToDocsRequests(%q)) = %d, want %d", tc.markdown, len(got), tc.wantLen)
			}
		})
	}
}

func TestRenderBodyAsText(t *testing.T) {
	tests := []struct {
		name string
		body *docs.Body
		want string
	}{
		{
			name: "nil_body",
			body: nil,
			want: "",
		},
		{
			name: "heading_bullet_and_styles",
			body: &docs.Body{
				Content: []*docs.StructuralElement{
					{
						Paragraph: &docs.Paragraph{
							ParagraphStyle: &docs.ParagraphStyle{NamedStyleType: "HEADING_2"},
							Elements: []*docs.ParagraphElement{
								{TextRun: &docs.TextRun{Content: "Section\n"}},
							},
						},
					},
					{
						Paragraph: &docs.Paragraph{
							Bullet: &docs.Bullet{NestingLevel: 1},
							Elements: []*docs.ParagraphElement{
								{
									TextRun: &docs.TextRun{
										Content:   "BoldItem",
										TextStyle: &docs.TextStyle{Bold: true},
									},
								},
								{
									TextRun: &docs.TextRun{
										Content: "DocsLink",
										TextStyle: &docs.TextStyle{
											Link: &docs.Link{Url: "https://example.com"},
										},
									},
								},
							},
						},
					},
				},
			},
			want: "## Section\n  * **BoldItem** [DocsLink](https://example.com) ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderBodyAsText(tc.body)
			if got != tc.want {
				t.Errorf("renderBodyAsText(%+v) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestTokenRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cachePath, err := TokenCacheFile()
	if err != nil {
		t.Fatalf("TokenCacheFile(HOME=%q) error = %v", home, err)
	}

	wantTok := &oauth2.Token{
		AccessToken:  "access-123",
		TokenType:    "Bearer",
		RefreshToken: "refresh-456",
	}
	if err := SaveToken(cachePath, wantTok); err != nil {
		t.Fatalf("SaveToken(%q, %+v) error = %v", cachePath, wantTok, err)
	}

	gotTok, err := TokenFromFile(cachePath)
	if err != nil {
		t.Fatalf("TokenFromFile(%q) error = %v", cachePath, err)
	}
	if diff := cmp.Diff(wantTok, gotTok, cmpopts.IgnoreUnexported(oauth2.Token{})); diff != "" {
		t.Errorf("TokenFromFile(%q) mismatch (-want +got):\n%s", cachePath, diff)
	}
}

func TestNewClientFromBearerTokenAndServices(t *testing.T) {
	var gotAuthHeader string
	baseHTTP := &http.Client{
		Transport: fakeRoundTripper(func(req *http.Request) (*http.Response, error) {
			gotAuthHeader = req.Header.Get("Authorization")
			return jsonHTTPResponse(req, http.StatusOK, `{"files":[{"id":"f1","name":"Notes.txt"}]}`), nil
		}),
	}

	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, baseHTTP)
	bearerClient := NewClientFromBearerToken(ctx, "stateless-token-xyz")

	svcs, err := NewServicesFromClient(ctx, bearerClient)
	if err != nil {
		t.Fatalf("NewServicesFromClient(ctx, bearerClient) error = %v", err)
	}
	if svcs.Drive == nil || svcs.Docs == nil || svcs.Sheets == nil || svcs.Slides == nil {
		t.Fatalf("NewServicesFromClient(ctx, bearerClient) = %+v, want non-nil Drive, Docs, Sheets, and Slides", svcs)
	}

	files, err := ListFiles(svcs.Drive, 5, "")
	if err != nil {
		t.Fatalf("ListFiles(svcs.Drive, 5, \"\") error = %v", err)
	}
	if len(files) != 1 || files[0].Id != "f1" {
		t.Errorf("ListFiles(svcs.Drive, 5, \"\") = %+v, want 1 file \"f1\"", files)
	}
	if gotAuthHeader != "Bearer stateless-token-xyz" {
		t.Errorf("NewClientFromBearerToken(ctx, %q) Authorization header = %q, want %q", "stateless-token-xyz", gotAuthHeader, "Bearer stateless-token-xyz")
	}
}

func TestDriveDocsSheetsAndSlidesServices(t *testing.T) {
	ctx := t.Context()
	driveSvc, docsSvc, sheetsSvc, slidesSvc := newFakeServices(t, ctx)

	files, err := ListFiles(driveSvc, 10, "")
	if err != nil {
		t.Fatalf("ListFiles(driveSvc, 10, \"\") error = %v", err)
	}
	if len(files) != 1 || files[0].Id != "f1" {
		t.Errorf("ListFiles(driveSvc, 10, \"\") = %+v, want 1 file with ID \"f1\"", files)
	}

	desc, err := DescribeFile(driveSvc, "f1")
	if err != nil {
		t.Fatalf("DescribeFile(driveSvc, \"f1\") error = %v", err)
	}
	if desc.Name != "Notes.txt" {
		t.Errorf("DescribeFile(driveSvc, \"f1\").Name = %q, want \"Notes.txt\"", desc.Name)
	}

	rawBytes, err := FetchFile(driveSvc, docsSvc, "f1", "", "")
	if err != nil {
		t.Fatalf("FetchFile(driveSvc, docsSvc, \"f1\", \"\", \"\") error = %v", err)
	}
	if string(rawBytes) != "raw-file-body" {
		t.Errorf("FetchFile(driveSvc, docsSvc, \"f1\", \"\", \"\") = %q, want \"raw-file-body\"", string(rawBytes))
	}

	exportedBytes, err := FetchFile(driveSvc, docsSvc, "gdoc1", "md", "")
	if err != nil {
		t.Fatalf("FetchFile(driveSvc, docsSvc, \"gdoc1\", \"md\", \"\") error = %v", err)
	}
	if string(exportedBytes) != "# Exported Markdown" {
		t.Errorf("FetchFile(driveSvc, docsSvc, \"gdoc1\", \"md\", \"\") = %q, want \"# Exported Markdown\"", string(exportedBytes))
	}

	tabBytes, err := FetchFile(driveSvc, docsSvc, "gdoc1", "", "t.sub")
	if err != nil {
		t.Fatalf("FetchFile(driveSvc, docsSvc, \"gdoc1\", \"\", \"t.sub\") error = %v", err)
	}
	if string(tabBytes) != "# SubTab Heading\n" {
		t.Errorf("FetchFile(driveSvc, docsSvc, \"gdoc1\", \"\", \"t.sub\") = %q, want \"# SubTab Heading\\n\"", string(tabBytes))
	}

	tabs, err := FetchTabs(docsSvc, "gdoc1")
	if err != nil {
		t.Fatalf("FetchTabs(docsSvc, \"gdoc1\") error = %v", err)
	}
	if len(tabs) != 1 || len(tabs[0].Children) != 1 || tabs[0].Children[0].TabID != "t.sub" {
		t.Errorf("FetchTabs(docsSvc, \"gdoc1\") = %+v, want nested tab \"t.sub\"", tabs)
	}

	createdDoc, err := CreateDocFromMarkdown(docsSvc, "Created Title", "# Header\n\nBody")
	if err != nil {
		t.Fatalf("CreateDocFromMarkdown(docsSvc, \"Created Title\", \"# Header\\n\\nBody\") error = %v", err)
	}
	if createdDoc.DocumentId != "new-doc-id" {
		t.Errorf("CreateDocFromMarkdown(docsSvc, \"Created Title\", ...) DocumentId = %q, want \"new-doc-id\"", createdDoc.DocumentId)
	}

	tmpFile := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(tmpFile, []byte("hello upload"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", tmpFile, err)
	}
	uploaded, err := UploadFile(driveSvc, tmpFile, "parent-1")
	if err != nil {
		t.Fatalf("UploadFile(driveSvc, %q, \"parent-1\") error = %v", tmpFile, err)
	}
	if uploaded.Id != "uploaded-1" {
		t.Errorf("UploadFile(driveSvc, %q, \"parent-1\").Id = %q, want \"uploaded-1\"", tmpFile, uploaded.Id)
	}

	comments, err := ListComments(driveSvc, "f1")
	if err != nil {
		t.Fatalf("ListComments(driveSvc, \"f1\") error = %v", err)
	}
	if len(comments) != 1 || comments[0].Content != "Looks good" {
		t.Errorf("ListComments(driveSvc, \"f1\") = %+v, want 1 comment \"Looks good\"", comments)
	}

	revisions, err := ListRevisions(driveSvc, "f1")
	if err != nil {
		t.Fatalf("ListRevisions(driveSvc, \"f1\") error = %v", err)
	}
	if len(revisions) != 1 || revisions[0].Id != "rev1" {
		t.Errorf("ListRevisions(driveSvc, \"f1\") = %+v, want 1 revision \"rev1\"", revisions)
	}

	sheetNames, err := ListSheets(sheetsSvc, "sp1")
	if err != nil {
		t.Fatalf("ListSheets(sheetsSvc, \"sp1\") error = %v", err)
	}
	if diff := cmp.Diff([]string{"Sheet1", "Metrics"}, sheetNames); diff != "" {
		t.Errorf("ListSheets(sheetsSvc, \"sp1\") mismatch (-want +got):\n%s", diff)
	}

	csvText, err := FetchSheetAsCSV(sheetsSvc, "sp1", "Sheet1")
	if err != nil {
		t.Fatalf("FetchSheetAsCSV(sheetsSvc, \"sp1\", \"Sheet1\") error = %v", err)
	}
	wantCSV := "Name,Description,Score\nAlice,\"Needs review, \"\"urgent\"\"\",95\nBob,\"Line 1\nLine 2\",88\n"
	if csvText != wantCSV {
		t.Errorf("FetchSheetAsCSV(sheetsSvc, \"sp1\", \"Sheet1\") = %q, want %q", csvText, wantCSV)
	}

	vals, err := FetchSheetRange(sheetsSvc, "sp1", "Sheet1", "A1:B2")
	if err != nil {
		t.Fatalf("FetchSheetRange(sheetsSvc, \"sp1\", \"Sheet1\", \"A1:B2\") error = %v", err)
	}
	if len(vals) != 3 {
		t.Errorf("len(FetchSheetRange(sheetsSvc, \"sp1\", \"Sheet1\", \"A1:B2\")) = %d, want 3", len(vals))
	}

	if err := UpdateSheetRange(sheetsSvc, "sp1", "Sheet1", "B2", [][]any{{"100"}}); err != nil {
		t.Fatalf("UpdateSheetRange(sheetsSvc, \"sp1\", \"Sheet1\", \"B2\", ...) error = %v", err)
	}

	presMD, err := FetchPresentationAsMarkdown(slidesSvc, "pres1")
	if err != nil {
		t.Fatalf("FetchPresentationAsMarkdown(slidesSvc, \"pres1\") error = %v", err)
	}
	wantPresMD := "# Q4 Roadmap\n\n## Slide 1\n\nKey Objectives\n| Goal | Owner |\n| Reliability | SRE |\n\n### Speaker Notes\n\nEmphasize latency targets.\n"
	if presMD != wantPresMD {
		t.Errorf("FetchPresentationAsMarkdown(slidesSvc, \"pres1\") = %q, want %q", presMD, wantPresMD)
	}

	createdPres, err := CreatePresentation(slidesSvc, "DevFest 2026")
	if err != nil {
		t.Fatalf("CreatePresentation(slidesSvc, \"DevFest 2026\") error = %v", err)
	}
	if createdPres.PresentationId != "new-pres-id" {
		t.Errorf("CreatePresentation(slidesSvc, \"DevFest 2026\").PresentationId = %q, want \"new-pres-id\"", createdPres.PresentationId)
	}

	batchResp, err := BatchUpdatePresentation(slidesSvc, "new-pres-id", []*slides.Request{
		{CreateSlide: &slides.CreateSlideRequest{ObjectId: "slide-2"}},
	})
	if err != nil {
		t.Fatalf("BatchUpdatePresentation(slidesSvc, \"new-pres-id\", ...) error = %v", err)
	}
	if batchResp.PresentationId != "new-pres-id" {
		t.Errorf("BatchUpdatePresentation(...).PresentationId = %q, want \"new-pres-id\"", batchResp.PresentationId)
	}
}

func TestResolveConvertMimeType(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		want    string
		wantErr bool
	}{
		{name: "empty", target: "", want: ""},
		{name: "slides", target: "slides", want: "application/vnd.google-apps.presentation"},
		{name: "presentation_uppercase", target: "Presentation", want: "application/vnd.google-apps.presentation"},
		{name: "docs", target: "docs", want: "application/vnd.google-apps.document"},
		{name: "sheets", target: "sheets", want: "application/vnd.google-apps.spreadsheet"},
		{name: "invalid", target: "pdf", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveConvertMimeType(tc.target)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveConvertMimeType(%q) error = %v, wantErr %v", tc.target, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ResolveConvertMimeType(%q) = %q, want %q", tc.target, got, tc.want)
			}
		})
	}
}

func TestUploadFileWithOptions(t *testing.T) {
	var capturedBody string
	var capturedFields string
	httpClient := &http.Client{
		Transport: fakeRoundTripper(func(req *http.Request) (*http.Response, error) {
			capturedFields = req.URL.Query().Get("fields")
			b, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			capturedBody = string(b)
			return jsonHTTPResponse(req, http.StatusOK, `{
				"id":"slides-123",
				"name":"Workshop Deck",
				"mimeType":"application/vnd.google-apps.presentation",
				"webViewLink":"https://docs.google.com/presentation/d/slides-123/edit"
			}`), nil
		}),
	}
	driveSvc, err := NewDriveService(t.Context(), httpClient)
	if err != nil {
		t.Fatalf("NewDriveService error = %v", err)
	}

	pptxPath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := os.WriteFile(pptxPath, []byte("PK-fake-pptx"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", pptxPath, err)
	}

	opts := UploadOptions{
		Name:     "Workshop Deck",
		ParentID: "folder-42",
		MimeType: "application/vnd.google-apps.presentation",
	}
	got, err := UploadFileWithOptions(driveSvc, pptxPath, opts)
	if err != nil {
		t.Fatalf("UploadFileWithOptions(driveSvc, %q, %+v) error = %v", pptxPath, opts, err)
	}
	if got.Id != "slides-123" || got.WebViewLink == "" {
		t.Errorf("UploadFileWithOptions returned %+v, want id=slides-123 and non-empty webViewLink", got)
	}
	if !strings.Contains(capturedFields, "webViewLink") {
		t.Errorf("upload fields query = %q, want webViewLink", capturedFields)
	}
	for _, wantSubstr := range []string{
		`"name":"Workshop Deck"`,
		`"mimeType":"application/vnd.google-apps.presentation"`,
		`"parents":["folder-42"]`,
		"Content-Type: application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"PK-fake-pptx",
	} {
		if !strings.Contains(capturedBody, wantSubstr) {
			t.Errorf("upload multipart body missing %q; got:\n%s", wantSubstr, capturedBody)
		}
	}
}

func TestUploadFileWithOptions_UpdateInPlaceAndStripExtension(t *testing.T) {
	var methods []string
	var paths []string
	var bodies []string

	httpClient := &http.Client{
		Transport: fakeRoundTripper(func(req *http.Request) (*http.Response, error) {
			methods = append(methods, req.Method)
			paths = append(paths, req.URL.Path)
			b, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			bodies = append(bodies, string(b))
			return jsonHTTPResponse(req, http.StatusOK, `{
				"id":"slides-existing",
				"name":"my-deck",
				"mimeType":"application/vnd.google-apps.presentation",
				"webViewLink":"https://docs.google.com/presentation/d/slides-existing/edit"
			}`), nil
		}),
	}
	driveSvc, err := NewDriveService(t.Context(), httpClient)
	if err != nil {
		t.Fatalf("NewDriveService error = %v", err)
	}

	pptxPath := filepath.Join(t.TempDir(), "my-deck.pptx")
	if err := os.WriteFile(pptxPath, []byte("PK-v2"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", pptxPath, err)
	}

	// 1. Convert with no explicit name strips .pptx extension.
	_, err = UploadFileWithOptions(driveSvc, pptxPath, UploadOptions{
		MimeType: "application/vnd.google-apps.presentation",
	})
	if err != nil {
		t.Fatalf("UploadFileWithOptions create convert error = %v", err)
	}
	if !strings.Contains(bodies[0], `"name":"my-deck"`) {
		t.Errorf("converted upload without Name should strip .pptx; body:\n%s", bodies[0])
	}

	// 2. Update in place with FileID uses PATCH to /files/{fileID}.
	got, err := UploadFileWithOptions(driveSvc, pptxPath, UploadOptions{
		FileID:   "slides-existing",
		MimeType: "application/vnd.google-apps.presentation",
	})
	if err != nil {
		t.Fatalf("UploadFileWithOptions update error = %v", err)
	}
	if got.Id != "slides-existing" {
		t.Errorf("updated Id = %q, want slides-existing", got.Id)
	}
	if methods[1] != http.MethodPatch || !strings.Contains(paths[1], "/files/slides-existing") {
		t.Errorf("in-place update used %s %s, want PATCH .../files/slides-existing", methods[1], paths[1])
	}
	if !strings.Contains(bodies[1], "Content-Type: application/vnd.openxmlformats-officedocument.presentationml.presentation") {
		t.Errorf("in-place update missing source Content-Type header; body:\n%s", bodies[1])
	}
}

func TestRenderPresentationAsMarkdown_SoftLineBreaks(t *testing.T) {
	pres := &slides.Presentation{
		Title: "Soft Breaks",
		Slides: []*slides.Page{
			{
				ObjectId: "s1",
				PageElements: []*slides.PageElement{
					{
						ObjectId: "el1",
						Shape: &slides.Shape{
							Text: &slides.TextContent{
								TextElements: []*slides.TextElement{
									{TextRun: &slides.TextRun{Content: "Line One\x0bLine Two\n"}},
								},
							},
						},
					},
					{
						ObjectId: "tbl1",
						Table: &slides.Table{
							TableRows: []*slides.TableRow{
								{
									TableCells: []*slides.TableCell{
										{
											Text: &slides.TextContent{
												TextElements: []*slides.TextElement{
													{TextRun: &slides.TextRun{Content: "Cell A\x0bWrapped\n"}},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	got := RenderPresentationAsMarkdown(pres)
	want := "# Soft Breaks\n\n## Slide 1\n\nLine One\nLine Two\n| Cell A Wrapped |\n\n"
	if got != want {
		t.Errorf("RenderPresentationAsMarkdown soft breaks = %q, want %q", got, want)
	}
}

func TestSlidesInspectReplaceAndNotes(t *testing.T) {
	var batchBodies []string
	httpClient := &http.Client{
		Transport: fakeRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && req.URL.Path == "/v1/presentations/p1":
				return jsonHTTPResponse(req, http.StatusOK, `{
					"presentationId":"p1",
					"title":"Workshop",
					"slides":[
						{
							"objectId":"slide-1",
							"pageElements":[
								{"objectId":"title-1","shape":{"text":{"textElements":[{"textRun":{"content":"Hello\u000bWorld\n"}}]}} },
								{"objectId":"img-1","image":{"contentUrl":"https://example.com/a.png"}}
							],
							"slideProperties":{"notesPage":{
								"notesProperties":{"speakerNotesObjectId":"notes-1"},
								"pageElements":[{"objectId":"notes-1","shape":{"text":{"textElements":[{"textRun":{"content":"Old note\n"}}]}}}]
							}}
						},
						{
							"objectId":"slide-2",
							"pageElements":[],
							"slideProperties":{"notesPage":{
								"notesProperties":{"speakerNotesObjectId":"notes-2"},
								"pageElements":[{"objectId":"notes-2","shape":{"text":{"textElements":[]}}}]
							}}
						}
					]
				}`), nil
			case req.Method == http.MethodPost && req.URL.Path == "/v1/presentations/p1:batchUpdate":
				b, _ := io.ReadAll(req.Body)
				batchBodies = append(batchBodies, string(b))
				return jsonHTTPResponse(req, http.StatusOK, `{"presentationId":"p1","replies":[{"replaceAllText":{"occurrencesChanged":3}}]}`), nil
			default:
				return jsonHTTPResponse(req, http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	slidesSvc, err := NewSlidesService(t.Context(), httpClient)
	if err != nil {
		t.Fatalf("NewSlidesService error = %v", err)
	}

	pres, err := FetchPresentation(slidesSvc, "p1")
	if err != nil {
		t.Fatalf("FetchPresentation error = %v", err)
	}
	outline := SummarizePresentation(pres)
	if len(outline.Slides) != 2 || outline.Slides[0].SpeakerNotesObjectID != "notes-1" {
		t.Fatalf("SummarizePresentation = %+v, want 2 slides with notes-1", outline)
	}
	if len(outline.Slides[0].Elements) != 2 || outline.Slides[0].Elements[0].Text != "Hello\nWorld" || outline.Slides[0].Elements[1].Kind != "image" {
		t.Errorf("Slide 1 elements = %+v, want Hello\\nWorld shape and image", outline.Slides[0].Elements)
	}

	changed, err := ReplacePresentationText(slidesSvc, "p1", "Old", "New", true)
	if err != nil {
		t.Fatalf("ReplacePresentationText error = %v", err)
	}
	if changed != 3 {
		t.Errorf("ReplacePresentationText changed = %d, want 3", changed)
	}

	// Updating non-empty notes sends deleteText + insertText.
	notesID, err := UpdateSlideNotes(slidesSvc, "p1", 1, "Updated note")
	if err != nil {
		t.Fatalf("UpdateSlideNotes(slide 1) error = %v", err)
	}
	if notesID != "notes-1" || !strings.Contains(batchBodies[1], `"deleteText"`) || !strings.Contains(batchBodies[1], `"Updated note"`) {
		t.Errorf("UpdateSlideNotes(slide 1) notesID=%q body=%s", notesID, batchBodies[1])
	}

	// Updating empty notes omits deleteText (Slides API rejects deleteText on empty shapes).
	notesID2, err := UpdateSlideNotes(slidesSvc, "p1", 2, "Brand new note")
	if err != nil {
		t.Fatalf("UpdateSlideNotes(slide 2) error = %v", err)
	}
	if notesID2 != "notes-2" || strings.Contains(batchBodies[2], `"deleteText"`) || !strings.Contains(batchBodies[2], `"Brand new note"`) {
		t.Errorf("UpdateSlideNotes(slide 2) notesID=%q body=%s", notesID2, batchBodies[2])
	}
}

type fakeRoundTripper func(*http.Request) (*http.Response, error)

func (f fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResponse(req *http.Request, status int, body string) *http.Response {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func newFakeServices(t *testing.T, ctx context.Context) (*drive.Service, *docs.Service, *sheets.Service, *slides.Service) {
	t.Helper()

	httpClient := &http.Client{
		Transport: fakeRoundTripper(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path
			q := req.URL.Query()
			switch {
			case req.Method == http.MethodGet && path == "/drive/v3/files":
				return jsonHTTPResponse(req, http.StatusOK, `{"files":[{"id":"f1","name":"Notes.txt"}]}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/f1" && q.Get("alt") == "media":
				return jsonHTTPResponse(req, http.StatusOK, "raw-file-body"), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/f1":
				return jsonHTTPResponse(req, http.StatusOK, `{"id":"f1","name":"Notes.txt","mimeType":"text/plain"}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/gdoc1/export":
				return jsonHTTPResponse(req, http.StatusOK, "# Exported Markdown"), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/gdoc1":
				return jsonHTTPResponse(req, http.StatusOK, `{"id":"gdoc1","name":"Doc","mimeType":"application/vnd.google-apps.document"}`), nil
			case req.Method == http.MethodPost && strings.Contains(path, "/drive/v3/files"):
				return jsonHTTPResponse(req, http.StatusOK, `{"id":"uploaded-1","name":"upload.txt"}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/f1/comments":
				return jsonHTTPResponse(req, http.StatusOK, `{"comments":[{"id":"c1","content":"Looks good","author":{"displayName":"Ada"}}]}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/f1/revisions":
				return jsonHTTPResponse(req, http.StatusOK, `{"revisions":[{"id":"rev1","modifiedTime":"2026-01-01T00:00:00Z"}]}`), nil
			case req.Method == http.MethodGet && path == "/v1/documents/gdoc1":
				return jsonHTTPResponse(req, http.StatusOK, `{
					"documentId":"gdoc1",
					"title":"Doc",
					"tabs":[{
						"tabProperties":{"tabId":"t.root","title":"Overview"},
						"childTabs":[{
							"tabProperties":{"tabId":"t.sub","title":"Details"},
							"documentTab":{"body":{"content":[{"paragraph":{"paragraphStyle":{"namedStyleType":"HEADING_1"},"elements":[{"textRun":{"content":"SubTab Heading\n"}}]}} ]}}
						}]
					}]
				}`), nil
			case req.Method == http.MethodPost && path == "/v1/documents":
				return jsonHTTPResponse(req, http.StatusOK, `{"documentId":"new-doc-id","title":"Created Title"}`), nil
			case req.Method == http.MethodPost && path == "/v1/documents/new-doc-id:batchUpdate":
				return jsonHTTPResponse(req, http.StatusOK, `{"documentId":"new-doc-id"}`), nil
			case req.Method == http.MethodGet && path == "/v4/spreadsheets/sp1":
				return jsonHTTPResponse(req, http.StatusOK, `{"sheets":[{"properties":{"title":"Sheet1"}},{"properties":{"title":"Metrics"}}]}`), nil
			case req.Method == http.MethodGet && strings.HasPrefix(path, "/v4/spreadsheets/sp1/values/"):
				return jsonHTTPResponse(req, http.StatusOK, `{"values":[["Name","Description","Score"],["Alice","Needs review, \"urgent\"","95"],["Bob","Line 1\nLine 2","88"]]}`), nil
			case req.Method == http.MethodPut && strings.HasPrefix(path, "/v4/spreadsheets/sp1/values/"):
				return jsonHTTPResponse(req, http.StatusOK, `{"updatedCells":1}`), nil
			case req.Method == http.MethodGet && path == "/v1/presentations/pres1":
				return jsonHTTPResponse(req, http.StatusOK, `{
					"presentationId":"pres1",
					"title":"Q4 Roadmap",
					"slides":[{
						"objectId":"s1",
						"pageElements":[
							{"objectId":"el1","shape":{"text":{"textElements":[{"textRun":{"content":"Key Objectives\n"}}]}} },
							{"objectId":"tbl1","table":{"tableRows":[
								{"tableCells":[{"text":{"textElements":[{"textRun":{"content":"Goal\n"}}]}},{"text":{"textElements":[{"textRun":{"content":"Owner\n"}}]}}]},
								{"tableCells":[{"text":{"textElements":[{"textRun":{"content":"Reliability\n"}}]}},{"text":{"textElements":[{"textRun":{"content":"SRE\n"}}]}}]}
							]}}
						],
						"slideProperties":{
							"notesPage":{
								"notesProperties":{"speakerNotesObjectId":"notes1"},
								"pageElements":[
									{"objectId":"notes1","shape":{"text":{"textElements":[{"textRun":{"content":"Emphasize latency targets.\n"}}]}}}
								]
							}
						}
					}]
				}`), nil
			case req.Method == http.MethodPost && path == "/v1/presentations":
				return jsonHTTPResponse(req, http.StatusOK, `{"presentationId":"new-pres-id","title":"DevFest 2026"}`), nil
			case req.Method == http.MethodPost && path == "/v1/presentations/new-pres-id:batchUpdate":
				return jsonHTTPResponse(req, http.StatusOK, `{"presentationId":"new-pres-id","replies":[{"createSlide":{"objectId":"slide-2"}}]}`), nil
			default:
				return jsonHTTPResponse(req, http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}

	svcs, err := NewServicesFromClient(ctx, httpClient)
	if err != nil {
		t.Fatalf("NewServicesFromClient(ctx, httpClient) error = %v", err)
	}
	return svcs.Drive, svcs.Docs, svcs.Sheets, svcs.Slides
}
