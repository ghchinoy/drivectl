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
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterTools_NativeToolsAndAnnotations(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := t.Context()

	fakeHTTP := newNativeToolsFakeHTTP()
	server := mcp.NewServer(&mcp.Implementation{Name: "native-tools-test"}, nil)
	if err := RegisterTools(
		ctx,
		server,
		func(context.Context) (*http.Client, error) { return fakeHTTP, nil },
		WithDiscoveryHTTPClient(fakeHTTP),
	); err != nil {
		t.Fatalf("RegisterTools(ctx, server, provider, WithDiscoveryHTTPClient) error = %v", err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect(ctx, serverTransport, nil) error = %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect(ctx, clientTransport, nil) error = %v", err)
	}
	defer clientSession.Close()

	listRes, err := clientSession.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("clientSession.ListTools(ctx, {}) error = %v", err)
	}

	toolsByName := make(map[string]*mcp.Tool, len(listRes.Tools))
	for _, tool := range listRes.Tools {
		toolsByName[tool.Name] = tool
		if tool.Annotations == nil {
			t.Errorf("tool %q has nil Annotations, want populated ToolAnnotations", tool.Name)
		}
	}

	// Verify expected annotation hints on read-only, additive, and mutating tools.
	assertToolHints(t, toolsByName, "drive.v3.files.list", true, false)
	assertToolHints(t, toolsByName, "docs.v1.documents.create", false, false)
	assertToolHints(t, toolsByName, "slides.v1.presentations.batchUpdate", false, true)
	assertToolHints(t, toolsByName, "drivectl_docs_export_markdown", true, false)
	assertToolHints(t, toolsByName, "drivectl_docs_create_from_markdown", false, false)
	assertToolHints(t, toolsByName, "drivectl_sheets_export_csv", true, false)
	assertToolHints(t, toolsByName, "drivectl_slides_get_markdown", true, false)
	assertToolHints(t, toolsByName, "drivectl_slides_inspect", true, false)
	assertToolHints(t, toolsByName, "drivectl_slides_update_notes", false, true)
	assertToolHints(t, toolsByName, "drivectl_slides_replace_text", false, true)

	// Exercise all 8 native tools end-to-end over in-memory MCP transport.
	calls := []struct {
		name    string
		args    map[string]any
		wantSub string
	}{
		{
			name:    "drivectl_docs_create_from_markdown",
			args:    map[string]any{"title": "Spec", "markdown": "# Heading\n\nBody"},
			wantSub: "doc-md-1",
		},
		{
			name:    "drivectl_docs_export_markdown",
			args:    map[string]any{"documentId": "doc-md-1"},
			wantSub: "Hello World",
		},
		{
			name:    "drivectl_docs_tabs",
			args:    map[string]any{"documentId": "doc-md-1"},
			wantSub: "Overview",
		},
		{
			name:    "drivectl_sheets_export_csv",
			args:    map[string]any{"spreadsheetId": "sheet-1", "sheetName": "Sheet1"},
			wantSub: `"a,b"`,
		},
		{
			name:    "drivectl_slides_get_markdown",
			args:    map[string]any{"presentationId": "pres-1"},
			wantSub: "Slide 1",
		},
		{
			name:    "drivectl_slides_inspect",
			args:    map[string]any{"presentationId": "pres-1", "slide": 1},
			wantSub: "notes-obj-1",
		},
		{
			name:    "drivectl_slides_update_notes",
			args:    map[string]any{"presentationId": "pres-1", "slide": 1, "notes": "Updated notes"},
			wantSub: "Updated notes",
		},
		{
			name:    "drivectl_slides_replace_text",
			args:    map[string]any{"presentationId": "pres-1", "find": "Old", "replace": "New", "matchCase": true},
			wantSub: `"occurrencesChanged": 2`,
		},
	}

	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      tc.name,
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("clientSession.CallTool(ctx, %q) error = %v", tc.name, err)
			}
			if res.IsError {
				t.Fatalf("clientSession.CallTool(ctx, %q) returned IsError=true: %+v", tc.name, res.Content)
			}
			if len(res.Content) == 0 {
				t.Fatalf("clientSession.CallTool(ctx, %q) returned empty Content", tc.name)
			}
			textContent, ok := res.Content[0].(*mcp.TextContent)
			if !ok || !strings.Contains(textContent.Text, tc.wantSub) {
				t.Errorf("clientSession.CallTool(ctx, %q) text = %+v, want substring %q", tc.name, res.Content[0], tc.wantSub)
			}
		})
	}
}

func assertToolHints(t *testing.T, tools map[string]*mcp.Tool, name string, wantReadOnly, wantDestructive bool) {
	t.Helper()
	tool, ok := tools[name]
	if !ok {
		t.Fatalf("tool %q not registered in ListTools", name)
	}
	if tool.Annotations == nil {
		t.Fatalf("tool %q Annotations = nil", name)
	}
	if tool.Annotations.ReadOnlyHint != wantReadOnly {
		t.Errorf("tool %q ReadOnlyHint = %t, want %t", name, tool.Annotations.ReadOnlyHint, wantReadOnly)
	}
	if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint != wantDestructive {
		t.Errorf("tool %q DestructiveHint = %v, want %t", name, tool.Annotations.DestructiveHint, wantDestructive)
	}
}

func newNativeToolsFakeHTTP() *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path
			switch {
			case strings.Contains(path, "/discovery/v1/apis/drive/v3/rest"):
				return newJSONResp(req, http.StatusOK, fakeDriveDiscoveryJSON), nil
			case strings.Contains(path, "/discovery/v1/apis/docs/v1/rest"):
				return newJSONResp(req, http.StatusOK, fakeDocsDiscoveryJSON), nil
			case strings.Contains(path, "/discovery/v1/apis/sheets/v4/rest"):
				return newJSONResp(req, http.StatusOK, fakeSheetsDiscoveryJSON), nil
			case strings.Contains(path, "/discovery/v1/apis/slides/v1/rest"):
				return newJSONResp(req, http.StatusOK, fakeSlidesDiscoveryJSON), nil
			case req.Method == http.MethodPost && path == "/v1/documents":
				return newJSONResp(req, http.StatusOK, `{"documentId":"doc-md-1","title":"Spec"}`), nil
			case req.Method == http.MethodPost && path == "/v1/documents/doc-md-1:batchUpdate":
				return newJSONResp(req, http.StatusOK, `{"documentId":"doc-md-1"}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/doc-md-1":
				return newJSONResp(req, http.StatusOK, `{"id":"doc-md-1","name":"Spec","mimeType":"application/vnd.google-apps.document"}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/doc-md-1/export":
				return newJSONResp(req, http.StatusOK, "# Spec\n\nHello World\n"), nil
			case req.Method == http.MethodGet && path == "/v1/documents/doc-md-1":
				return newJSONResp(req, http.StatusOK, `{"documentId":"doc-md-1","title":"Spec","body":{"content":[{"paragraph":{"elements":[{"textRun":{"content":"Hello World\n"}}]}}]},"tabs":[{"tabProperties":{"tabId":"t1","title":"Overview"}}]}`), nil
			case req.Method == http.MethodGet && strings.Contains(path, "/v4/spreadsheets/sheet-1/values/"):
				return newJSONResp(req, http.StatusOK, `{"values":[["col1","col2"],["a,b","ok"]]}`), nil
			case req.Method == http.MethodGet && path == "/v1/presentations/pres-1":
				return newJSONResp(req, http.StatusOK, `{
					"presentationId":"pres-1",
					"title":"Workshop",
					"slides":[{
						"objectId":"slide-1",
						"pageElements":[{"objectId":"shape-1","shape":{"text":{"textElements":[{"textRun":{"content":"Welcome\n"}}]}}}],
						"slideProperties":{"notesPage":{"notesProperties":{"speakerNotesObjectId":"notes-obj-1"},"pageElements":[{"objectId":"notes-obj-1","shape":{"text":{"textElements":[{"textRun":{"content":"Initial notes\n"}}]}}}]}}
					}]
				}`), nil
			case req.Method == http.MethodPost && path == "/v1/presentations/pres-1:batchUpdate":
				return newJSONResp(req, http.StatusOK, `{"presentationId":"pres-1","replies":[{"replaceAllText":{"occurrencesChanged":2}}]}`), nil
			default:
				return newJSONResp(req, http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
}
