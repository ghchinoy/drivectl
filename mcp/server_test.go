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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServer_ResourcesAndDynamicTools(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := t.Context()

	fakeHTTP := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.Contains(req.URL.Path, "/discovery/v1/apis/drive/v3/rest"):
				return newJSONResp(req, http.StatusOK, fakeDriveDiscoveryJSON), nil
			case strings.Contains(req.URL.Path, "/discovery/v1/apis/docs/v1/rest"):
				return newJSONResp(req, http.StatusOK, fakeDocsDiscoveryJSON), nil
			case strings.Contains(req.URL.Path, "/discovery/v1/apis/sheets/v4/rest"):
				return newJSONResp(req, http.StatusOK, fakeSheetsDiscoveryJSON), nil
			case req.Method == http.MethodGet && req.URL.Path == "/drive/v3/files":
				return newJSONResp(req, http.StatusOK, `{"files":[{"id":"123","name":"Design.md"}]}`), nil
			case req.Method == http.MethodPost && req.URL.Path == "/v1/documents":
				return newJSONResp(req, http.StatusOK, `{"documentId":"doc-42","title":"Quarterly Plan"}`), nil
			default:
				return newJSONResp(req, http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}

	origDiscoveryClient := discoveryHTTPClient
	origAPIFactory := apiClientFactory
	discoveryHTTPClient = fakeHTTP
	apiClientFactory = func(context.Context) (*http.Client, error) {
		return fakeHTTP, nil
	}
	t.Cleanup(func() {
		discoveryHTTPClient = origDiscoveryClient
		apiClientFactory = origAPIFactory
	})

	server := mcp.NewServer(&mcp.Implementation{Name: "drivectl"}, nil)
	if err := registerToolsAndResources(ctx, server); err != nil {
		t.Fatalf("registerToolsAndResources(ctx, server) error = %v", err)
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

	resList, err := clientSession.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		t.Fatalf("clientSession.ListResources(ctx, {}) error = %v", err)
	}
	if len(resList.Resources) != 2 {
		t.Errorf("len(clientSession.ListResources(ctx, {}).Resources) = %d, want 2", len(resList.Resources))
	}

	readRes, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "embedded:drive-query-cheat-sheet",
	})
	if err != nil {
		t.Fatalf("clientSession.ReadResource(ctx, \"embedded:drive-query-cheat-sheet\") error = %v", err)
	}
	if len(readRes.Contents) != 1 || !strings.Contains(readRes.Contents[0].Text, "mimeType=") {
		t.Errorf("clientSession.ReadResource(ctx, \"embedded:drive-query-cheat-sheet\") = %+v, want cheat sheet text", readRes.Contents)
	}

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "drive.v3.files.list",
		Arguments: map[string]any{"pageSize": 5},
	})
	if err != nil {
		t.Fatalf("clientSession.CallTool(ctx, \"drive.v3.files.list\") error = %v", err)
	}
	if callRes.IsError {
		t.Fatalf("clientSession.CallTool(ctx, \"drive.v3.files.list\") returned IsError=true: %+v", callRes.Content)
	}

	docCallRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "docs.v1.documents.create",
		Arguments: map[string]any{
			"payload": map[string]any{"title": "Quarterly Plan"},
		},
	})
	if err != nil {
		t.Fatalf("clientSession.CallTool(ctx, \"docs.v1.documents.create\") error = %v", err)
	}
	if docCallRes.IsError {
		t.Fatalf("clientSession.CallTool(ctx, \"docs.v1.documents.create\") returned IsError=true: %+v", docCallRes.Content)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newJSONResp(req *http.Request, status int, body string) *http.Response {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

const fakeDriveDiscoveryJSON = `{
  "name": "drive",
  "version": "v3",
  "baseUrl": "https://www.googleapis.com/drive/v3/",
  "schemas": {
    "File": {
      "id": "File",
      "type": "object",
      "properties": {
        "name": {"type": "string"}
      }
    }
  },
  "resources": {
    "files": {
      "methods": {
        "list": {
          "id": "drive.files.list",
          "httpMethod": "GET",
          "path": "files",
          "parameters": {
            "pageSize": {"type": "integer", "location": "query", "default": "100"},
            "supportsAllDrives": {"type": "boolean", "location": "query", "default": "false"}
          }
        },
        "get": {
          "id": "drive.files.get",
          "httpMethod": "GET",
          "path": "files/{fileId}",
          "parameters": {
            "fileId": {"type": "string", "location": "path", "required": true}
          }
        },
        "create": {
          "id": "drive.files.create",
          "httpMethod": "POST",
          "path": "files",
          "request": {"$ref": "File"}
        },
        "update": {
          "id": "drive.files.update",
          "httpMethod": "PATCH",
          "path": "files/{fileId}",
          "parameters": {
            "fileId": {"type": "string", "location": "path", "required": true}
          },
          "request": {"$ref": "File"}
        }
      }
    }
  }
}`

const fakeDocsDiscoveryJSON = `{
  "name": "docs",
  "version": "v1",
  "baseUrl": "https://docs.googleapis.com/",
  "schemas": {
    "Document": {
      "id": "Document",
      "type": "object",
      "properties": {
        "title": {"type": "string"},
        "tabs": {"type": "array", "items": {"$ref": "Tab"}}
      }
    },
    "Tab": {
      "id": "Tab",
      "type": "object",
      "properties": {
        "childTabs": {"type": "array", "items": {"$ref": "Tab"}}
      }
    },
    "BatchUpdateDocumentRequest": {
      "id": "BatchUpdateDocumentRequest",
      "type": "object",
      "properties": {}
    }
  },
  "resources": {
    "documents": {
      "methods": {
        "create": {
          "id": "docs.documents.create",
          "httpMethod": "POST",
          "path": "v1/documents",
          "request": {"$ref": "Document"}
        },
        "get": {
          "id": "docs.documents.get",
          "httpMethod": "GET",
          "path": "v1/documents/{documentId}",
          "parameters": {
            "documentId": {"type": "string", "location": "path", "required": true},
            "includeTabsContent": {"type": "boolean", "location": "query", "default": "false"}
          }
        },
        "batchUpdate": {
          "id": "docs.documents.batchUpdate",
          "httpMethod": "POST",
          "path": "v1/documents/{documentId}:batchUpdate",
          "parameters": {
            "documentId": {"type": "string", "location": "path", "required": true}
          },
          "request": {"$ref": "BatchUpdateDocumentRequest"}
        }
      }
    }
  }
}`

const fakeSheetsDiscoveryJSON = `{
  "name": "sheets",
  "version": "v4",
  "baseUrl": "https://sheets.googleapis.com/",
  "schemas": {
    "Spreadsheet": {
      "id": "Spreadsheet",
      "type": "object",
      "properties": {}
    },
    "ValueRange": {
      "id": "ValueRange",
      "type": "object",
      "properties": {
        "values": {"type": "array", "items": {"type": "any"}}
      }
    }
  },
  "resources": {
    "spreadsheets": {
      "methods": {
        "create": {
          "id": "sheets.spreadsheets.create",
          "httpMethod": "POST",
          "path": "v4/spreadsheets",
          "request": {"$ref": "Spreadsheet"}
        },
        "get": {
          "id": "sheets.spreadsheets.get",
          "httpMethod": "GET",
          "path": "v4/spreadsheets/{spreadsheetId}",
          "parameters": {
            "spreadsheetId": {"type": "string", "location": "path", "required": true}
          }
        }
      },
      "resources": {
        "values": {
          "methods": {
            "get": {
              "id": "sheets.spreadsheets.values.get",
              "httpMethod": "GET",
              "path": "v4/spreadsheets/{spreadsheetId}/values/{range}",
              "parameters": {
                "spreadsheetId": {"type": "string", "location": "path", "required": true},
                "range": {"type": "string", "location": "path", "required": true}
              }
            },
            "update": {
              "id": "sheets.spreadsheets.values.update",
              "httpMethod": "PUT",
              "path": "v4/spreadsheets/{spreadsheetId}/values/{range}",
              "parameters": {
                "spreadsheetId": {"type": "string", "location": "path", "required": true},
                "range": {"type": "string", "location": "path", "required": true}
              },
              "request": {"$ref": "ValueRange"}
            }
          }
        }
      }
    }
  }
}`
