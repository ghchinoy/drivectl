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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := t.Context()

	origClient := client
	client = newFakeCmdHTTPClient()
	t.Cleanup(func() {
		client = origClient
		OutputFormat = ""
	})

	mdFile := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(mdFile, []byte("# Title\n\nParagraph text."), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", mdFile, err)
	}

	recipeFile := filepath.Join(t.TempDir(), "recipe.json")
	recipeContent := `{"name":"Test Recipe","description":"Runs list","steps":[["list","--limit","2","-O","json"]]}`
	if err := os.WriteFile(recipeFile, []byte(recipeContent), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", recipeFile, err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "list_default",
			args: []string{"list", "--limit", "5"},
		},
		{
			name: "list_json",
			args: []string{"list", "-O", "json"},
		},
		{
			name: "describe_file",
			args: []string{"describe", "f1"},
		},
		{
			name: "get_file_json",
			args: []string{"get", "f1", "-O", "json"},
		},
		{
			name: "comments_list",
			args: []string{"comments", "f1"},
		},
		{
			name: "revisions_list",
			args: []string{"revisions", "f1"},
		},
		{
			name: "docs_tabs",
			args: []string{"docs", "tabs", "gdoc1"},
		},
		{
			name: "docs_create",
			args: []string{"docs", "create", "My Doc", mdFile},
		},
		{
			name: "sheets_list",
			args: []string{"sheets", "list", "sp1"},
		},
		{
			name: "sheets_get_csv",
			args: []string{"sheets", "get", "sp1", "--sheet", "Sheet1"},
		},
		{
			name: "sheets_get_range",
			args: []string{"sheets", "get-range", "sp1", "--sheet", "Sheet1", "--range", "A1:B2"},
		},
		{
			name: "sheets_update_range",
			args: []string{"sheets", "update-range", "sp1", "Done", "--sheet", "Sheet1", "--range", "B2"},
		},
		{
			name: "upload_file",
			args: []string{"upload", mdFile},
		},
		{
			name: "call_dynamic_endpoint",
			args: []string{"call", "drive.v3.files.list", "--payload", `{"pageSize":2}`},
		},
		{
			name:    "call_invalid_endpoint_format",
			args:    []string{"call", "invalid-format"},
			wantErr: true,
		},
		{
			name: "run_recipe",
			args: []string{"run", recipeFile},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			OutputFormat = ""
			payloadData = ""
			outputFile = ""
			format = ""
			tabID = ""

			rootCmd.SetArgs(tc.args)
			err := ExecuteContext(ctx)
			if (err != nil) != tc.wantErr {
				t.Errorf("ExecuteContext(%v) error = %v, wantErr %v", tc.args, err, tc.wantErr)
			}
		})
	}
}

func TestCopySecretFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.json")
	dst := filepath.Join(dir, "dst.json")
	want := `{"installed":{"client_id":"test"}}`

	if err := os.WriteFile(src, []byte(want), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", src, err)
	}
	if err := copySecretFile(src, dst); err != nil {
		t.Fatalf("copySecretFile(%q, %q) error = %v", src, dst, err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", dst, err)
	}
	if string(got) != want {
		t.Errorf("copySecretFile(%q, %q) content = %q, want %q", src, dst, string(got), want)
	}
}

type cmdRoundTripFunc func(*http.Request) (*http.Response, error)

func (f cmdRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newFakeCmdHTTPClient() *http.Client {
	return &http.Client{
		Transport: cmdRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path
			q := req.URL.Query()
			switch {
			case strings.Contains(path, "/discovery/v1/apis/drive/v3/rest"):
				return cmdJSONResp(req, http.StatusOK, `{
					"name":"drive",
					"version":"v3",
					"baseUrl":"https://www.googleapis.com/drive/v3/",
					"resources":{"files":{"methods":{"list":{"id":"drive.files.list","httpMethod":"GET","path":"files"}}}}
				}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files":
				return cmdJSONResp(req, http.StatusOK, `{"files":[{"id":"f1","name":"Notes.txt"}]}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/f1" && q.Get("alt") == "media":
				return cmdJSONResp(req, http.StatusOK, "hello file"), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/f1":
				return cmdJSONResp(req, http.StatusOK, `{"id":"f1","name":"Notes.txt","mimeType":"text/plain"}`), nil
			case req.Method == http.MethodPost && strings.Contains(path, "/drive/v3/files"):
				return cmdJSONResp(req, http.StatusOK, `{"id":"up1","name":"doc.md"}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/f1/comments":
				return cmdJSONResp(req, http.StatusOK, `{"comments":[{"id":"c1","content":"Great","author":{"displayName":"Bob"},"replies":[{"content":"Thanks","author":{"displayName":"Alice"}}]}]}`), nil
			case req.Method == http.MethodGet && path == "/drive/v3/files/f1/revisions":
				return cmdJSONResp(req, http.StatusOK, `{"revisions":[{"id":"r1","modifiedTime":"2026-01-01T00:00:00Z","lastModifyingUser":{"displayName":"Bob"}}]}`), nil
			case req.Method == http.MethodGet && path == "/v1/documents/gdoc1":
				return cmdJSONResp(req, http.StatusOK, `{"documentId":"gdoc1","tabs":[{"tabProperties":{"tabId":"t1","title":"Main"}}]}`), nil
			case req.Method == http.MethodPost && path == "/v1/documents":
				return cmdJSONResp(req, http.StatusOK, `{"documentId":"doc1","title":"My Doc"}`), nil
			case req.Method == http.MethodPost && path == "/v1/documents/doc1:batchUpdate":
				return cmdJSONResp(req, http.StatusOK, `{"documentId":"doc1"}`), nil
			case req.Method == http.MethodGet && path == "/v4/spreadsheets/sp1":
				return cmdJSONResp(req, http.StatusOK, `{"sheets":[{"properties":{"title":"Sheet1"}}]}`), nil
			case req.Method == http.MethodGet && strings.HasPrefix(path, "/v4/spreadsheets/sp1/values/"):
				return cmdJSONResp(req, http.StatusOK, `{"values":[["Col1","Col2"],["Val1","Val2"]]}`), nil
			case req.Method == http.MethodPut && strings.HasPrefix(path, "/v4/spreadsheets/sp1/values/"):
				return cmdJSONResp(req, http.StatusOK, `{"updatedCells":1}`), nil
			default:
				return cmdJSONResp(req, http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
}

func cmdJSONResp(req *http.Request, status int, body string) *http.Response {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
