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

package discovery

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/jsonschema-go/jsonschema"
)

func TestFindMethod(t *testing.T) {
	doc := sampleDiscoveryDoc()

	tests := []struct {
		name         string
		doc          *RestDescription
		resourcePath []string
		methodName   string
		wantID       string
		wantErr      bool
	}{
		{
			name:         "top_level_method",
			doc:          doc,
			resourcePath: []string{"files"},
			methodName:   "list",
			wantID:       "drive.files.list",
		},
		{
			name:         "nested_subresource_method",
			doc:          doc,
			resourcePath: []string{"spreadsheets", "values"},
			methodName:   "get",
			wantID:       "sheets.spreadsheets.values.get",
		},
		{
			name:         "nil_doc",
			doc:          nil,
			resourcePath: []string{"files"},
			methodName:   "list",
			wantErr:      true,
		},
		{
			name:         "empty_resource_path",
			doc:          doc,
			resourcePath: nil,
			methodName:   "list",
			wantErr:      true,
		},
		{
			name:         "unknown_resource",
			doc:          doc,
			resourcePath: []string{"nonexistent"},
			methodName:   "list",
			wantErr:      true,
		},
		{
			name:         "unknown_subresource",
			doc:          doc,
			resourcePath: []string{"spreadsheets", "unknown"},
			methodName:   "get",
			wantErr:      true,
		},
		{
			name:         "unknown_method",
			doc:          doc,
			resourcePath: []string{"files"},
			methodName:   "deleteAll",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FindMethod(tc.doc, tc.resourcePath, tc.methodName)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FindMethod(doc, %v, %q) error = %v, wantErr %v", tc.resourcePath, tc.methodName, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got.ID != tc.wantID {
				t.Errorf("FindMethod(doc, %v, %q).ID = %q, want %q", tc.resourcePath, tc.methodName, got.ID, tc.wantID)
			}
		})
	}
}

func TestBuildRequest(t *testing.T) {
	ctx := t.Context()
	doc := sampleDiscoveryDoc()

	tests := []struct {
		name             string
		method           *RestMethod
		parsed           map[string]any
		unwrapPayloadKey bool
		wantMethod       string
		wantURL          string
		wantBody         string
		wantErr          bool
	}{
		{
			name:       "get_with_query_params",
			method:     mustFindMethod(t, doc, []string{"files"}, "list"),
			parsed:     map[string]any{"pageSize": 10},
			wantMethod: http.MethodGet,
			wantURL:    "https://www.googleapis.com/drive/v3/files?pageSize=10",
		},
		{
			name:             "post_with_path_and_unwrapped_payload",
			method:           mustFindMethod(t, doc, []string{"files"}, "update"),
			parsed:           map[string]any{"fileId": "abc/123", "payload": map[string]any{"name": "updated.txt"}},
			unwrapPayloadKey: true,
			wantMethod:       http.MethodPatch,
			wantURL:          "https://www.googleapis.com/drive/v3/files/abc%2F123",
			wantBody:         `{"name":"updated.txt"}`,
		},
		{
			name:       "missing_required_path_parameter",
			method:     mustFindMethod(t, doc, []string{"files"}, "update"),
			parsed:     map[string]any{"name": "updated.txt"},
			wantMethod: http.MethodPatch,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := BuildRequest(ctx, doc, tc.method, tc.parsed, tc.unwrapPayloadKey)
			if (err != nil) != tc.wantErr {
				t.Fatalf("BuildRequest(ctx, doc, %q, %v, %v) error = %v, wantErr %v", tc.method.ID, tc.parsed, tc.unwrapPayloadKey, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if req.Method != tc.wantMethod {
				t.Errorf("BuildRequest(ctx, doc, %q, %v, %v).Method = %q, want %q", tc.method.ID, tc.parsed, tc.unwrapPayloadKey, req.Method, tc.wantMethod)
			}
			if req.URL.String() != tc.wantURL {
				t.Errorf("BuildRequest(ctx, doc, %q, %v, %v).URL = %q, want %q", tc.method.ID, tc.parsed, tc.unwrapPayloadKey, req.URL.String(), tc.wantURL)
			}
			if tc.wantBody != "" {
				bodyBytes, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("io.ReadAll(req.Body) error = %v", err)
				}
				if gotBody := string(bodyBytes); gotBody != tc.wantBody {
					t.Errorf("BuildRequest(ctx, doc, %q, %v, %v) body = %q, want %q", tc.method.ID, tc.parsed, tc.unwrapPayloadKey, gotBody, tc.wantBody)
				}
			}
		})
	}
}

func TestFetchDiscoveryDocument_CacheAndNetwork(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := t.Context()

	calls := 0
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			body := `{"name":"drive","version":"v3","baseUrl":"https://www.googleapis.com/drive/v3/"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	doc1, err := FetchDiscoveryDocument(ctx, client, "drive", "v3")
	if err != nil {
		t.Fatalf("FetchDiscoveryDocument(ctx, client, \"drive\", \"v3\") error = %v", err)
	}
	if doc1.Name != "drive" || doc1.Version != "v3" {
		t.Errorf("FetchDiscoveryDocument(ctx, client, \"drive\", \"v3\") = (%q, %q), want (\"drive\", \"v3\")", doc1.Name, doc1.Version)
	}

	doc2, err := FetchDiscoveryDocument(ctx, client, "drive", "v3")
	if err != nil {
		t.Fatalf("FetchDiscoveryDocument(ctx, client, \"drive\", \"v3\") cached error = %v", err)
	}
	if calls != 1 {
		t.Errorf("FetchDiscoveryDocument(ctx, client, \"drive\", \"v3\") network calls = %d, want 1 (second call should hit cache)", calls)
	}
	if diff := cmp.Diff(doc1, doc2); diff != "" {
		t.Errorf("FetchDiscoveryDocument(ctx, client, \"drive\", \"v3\") cached mismatch (-want +got):\n%s", diff)
	}
}

func TestMethodToSchema_ResolvesDefaultsAndRecursiveRefs(t *testing.T) {
	doc := sampleDiscoveryDoc()
	method := mustFindMethod(t, doc, []string{"documents"}, "create")

	schema := MethodToSchema(method, doc)
	if schema.Type != "object" {
		t.Fatalf("MethodToSchema(%q, doc).Type = %q, want \"object\"", method.ID, schema.Type)
	}

	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		t.Fatalf("MethodToSchema(%q, doc).Resolve(ValidateDefaults) error = %v", method.ID, err)
	}

	validInstance := map[string]any{
		"includeTabsContent": true,
		"pageSize":           100,
		"payload": map[string]any{
			"title": "Test Doc",
			"tabs": []any{
				map[string]any{
					"tabId": "t1",
				},
			},
		},
	}
	if err := resolved.Validate(validInstance); err != nil {
		t.Errorf("resolved.Validate(%v) error = %v, want nil", validInstance, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mustFindMethod(t *testing.T, doc *RestDescription, resourcePath []string, methodName string) *RestMethod {
	t.Helper()
	m, err := FindMethod(doc, resourcePath, methodName)
	if err != nil {
		t.Fatalf("FindMethod(doc, %v, %q) failed: %v", resourcePath, methodName, err)
	}
	return m
}

func sampleDiscoveryDoc() *RestDescription {
	return &RestDescription{
		Name:    "drive",
		Version: "v3",
		BaseURL: "https://www.googleapis.com/drive/v3/",
		Schemas: map[string]JSONSchema{
			"File": {
				ID:   "File",
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"name":  {Type: "string", Description: "File name"},
					"extra": {Type: "any", Description: "Arbitrary metadata"},
				},
			},
			"Document": {
				ID:   "Document",
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"title": {Type: "string"},
					"tabs": {
						Type:  "array",
						Items: &JSONSchemaProperty{Ref: "Tab"},
					},
				},
			},
			"Tab": {
				ID:   "Tab",
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"tabId": {Type: "string"},
					"childTabs": {
						Type:  "array",
						Items: &JSONSchemaProperty{Ref: "Tab"},
					},
				},
			},
		},
		Resources: map[string]RestResource{
			"files": {
				Methods: map[string]RestMethod{
					"list": {
						ID:         "drive.files.list",
						HTTPMethod: http.MethodGet,
						Path:       "files",
						Parameters: map[string]MethodParameter{
							"pageSize": {
								Type:     "integer",
								Location: "query",
								Default:  "100",
							},
						},
					},
					"update": {
						ID:         "drive.files.update",
						HTTPMethod: http.MethodPatch,
						Path:       "files/{fileId}",
						Parameters: map[string]MethodParameter{
							"fileId": {
								Type:     "string",
								Location: "path",
								Required: true,
							},
						},
						Request: &SchemaRef{Ref: "File"},
					},
				},
			},
			"spreadsheets": {
				Resources: map[string]RestResource{
					"values": {
						Methods: map[string]RestMethod{
							"get": {
								ID:         "sheets.spreadsheets.values.get",
								HTTPMethod: http.MethodGet,
								Path:       "v4/spreadsheets/{spreadsheetId}/values/{range}",
							},
						},
					},
				},
			},
			"documents": {
				Methods: map[string]RestMethod{
					"create": {
						ID:         "docs.documents.create",
						HTTPMethod: http.MethodPost,
						Path:       "v1/documents",
						Parameters: map[string]MethodParameter{
							"includeTabsContent": {
								Type:     "boolean",
								Location: "query",
								Default:  "false",
							},
							"pageSize": {
								Type:     "integer",
								Location: "query",
								Default:  "50",
							},
							"spaces": {
								Type:     "string",
								Location: "query",
								Default:  "drive",
								Enum:     []string{"drive", "appDataFolder"},
							},
						},
						Request: &SchemaRef{Ref: "Document"},
					},
				},
			},
		},
	}
}

var _ = bytes.NewReader
