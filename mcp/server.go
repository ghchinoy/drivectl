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

// Package mcp implements the Model Context Protocol (MCP) server for drivectl.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ghchinoy/drivectl/pkg/discovery"
	"github.com/ghchinoy/drivectl/pkg/drive"
)

const driveQueryCheatSheet = `
- "mimeType='application/vnd.google-apps.folder'"
- "name contains 'meeting notes'"
- "modifiedTime > '2025-01-01T00:00:00Z'"
- "trashed = false"
`

const a1NotationCheatSheet = `
A1 notation is a way to specify a cell or a range of cells in a spreadsheet. It consists of the column letter(s) followed by the row number.

Examples:
- A1 refers to the cell at the intersection of column A and row 1.
- A1:B2 refers to the range of cells from A1 to B2.
- Sheet1!A1:B2 refers to the range A1:B2 on the sheet named "Sheet1".
`

// Dynamic tools to expose via MCP. Format: service.version.resource.method.
var dynamicTools = []string{
	"drive.v3.files.list",
	"drive.v3.files.get",
	"drive.v3.files.create",
	"drive.v3.files.update",
	"docs.v1.documents.create",
	"docs.v1.documents.get",
	"docs.v1.documents.batchUpdate",
	"sheets.v4.spreadsheets.create",
	"sheets.v4.spreadsheets.get",
	"sheets.v4.spreadsheets.values.get",
	"sheets.v4.spreadsheets.values.update",
	"slides.v1.presentations.get",
}

// ClientProvider returns an authenticated *http.Client for executing Google Workspace API requests.
type ClientProvider func(ctx context.Context) (*http.Client, error)

type registerConfig struct {
	discoveryClient *http.Client
}

// RegisterOption configures optional settings for RegisterTools.
type RegisterOption func(*registerConfig)

// WithDiscoveryHTTPClient overrides the HTTP client used to fetch Google Discovery Documents.
func WithDiscoveryHTTPClient(client *http.Client) RegisterOption {
	return func(cfg *registerConfig) {
		cfg.discoveryClient = client
	}
}

var (
	discoveryHTTPClient = http.DefaultClient
	apiClientFactory    = newHTTPClient
)

// Start starts the MCP server over HTTP (when httpAddr is non-empty) or stdio.
func Start(ctx context.Context, _ *cobra.Command, httpAddr string) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "drivectl"}, nil)

	if err := registerToolsAndResources(ctx, server); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	if httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		srv := &http.Server{
			Addr:    httpAddr,
			Handler: handler,
		}
		go func() {
			<-ctx.Done()
			_ = srv.Shutdown(ctx)
		}()
		log.Printf("MCP handler listening at %s", httpAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}

	logFile, err := os.OpenFile("drivectl-mcp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	t := &mcp.LoggingTransport{
		Transport: &mcp.StdioTransport{},
		Writer:    logFile,
	}
	if err := server.Run(ctx, t); err != nil {
		log.Printf("Server failed: %v", err)
		return err
	}

	return nil
}

func newHTTPClient(ctx context.Context) (*http.Client, error) {
	viper.AutomaticEnv()
	secretFile := viper.GetString("secret-file")
	if secretFile == "" {
		configDir, err := drive.ConfigDir()
		if err == nil {
			secretFile = filepath.Join(configDir, "client_secret.json")
		}
	}
	noBrowserAuth := viper.GetBool("no-browser-auth")
	client, err := drive.NewOAuthClient(ctx, secretFile, noBrowserAuth)
	if err != nil {
		return nil, fmt.Errorf("could not create oauth client: %w", err)
	}
	return client, nil
}

func driveQueryCheatSheetHandler(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     driveQueryCheatSheet,
			},
		},
	}, nil
}

func a1NotationCheatSheetHandler(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     a1NotationCheatSheet,
			},
		},
	}, nil
}

func registerToolsAndResources(ctx context.Context, server *mcp.Server) error {
	return RegisterTools(ctx, server, apiClientFactory, WithDiscoveryHTTPClient(discoveryHTTPClient))
}

// RegisterTools mounts drivectl's embedded resources and Discovery-backed Google Workspace
// tools onto an existing *mcp.Server using the provided ClientProvider.
// It does not mutate global state or open local log files.
func RegisterTools(ctx context.Context, server *mcp.Server, clientProvider ClientProvider, opts ...RegisterOption) error {
	if server == nil {
		return errors.New("mcp server cannot be nil")
	}
	if clientProvider == nil {
		clientProvider = newHTTPClient
	}

	cfg := &registerConfig{
		discoveryClient: discoveryHTTPClient,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	server.AddResource(&mcp.Resource{
		Name:        "drive-query-cheat-sheet",
		Description: "A cheat sheet of example Google Drive query examples.",
		MIMEType:    "text/plain",
		URI:         "embedded:drive-query-cheat-sheet",
	}, driveQueryCheatSheetHandler)

	server.AddResource(&mcp.Resource{
		Name:        "a1-notation-cheat-sheet",
		Description: "A cheat sheet of example A1 notation for Google Sheets.",
		MIMEType:    "text/plain",
		URI:         "embedded:a1-notation-cheat-sheet",
	}, a1NotationCheatSheetHandler)

	docsCache := make(map[string]*discovery.RestDescription)

	for _, endpoint := range dynamicTools {
		registerDynamicEndpoint(ctx, server, cfg.discoveryClient, clientProvider, docsCache, endpoint)
	}

	return nil
}

func registerDynamicEndpoint(
	ctx context.Context,
	server *mcp.Server,
	discClient *http.Client,
	clientProvider ClientProvider,
	docsCache map[string]*discovery.RestDescription,
	endpoint string,
) {
	parts := strings.Split(endpoint, ".")
	serviceName := parts[0]
	version := parts[1]
	cacheKey := serviceName + "_" + version

	if _, ok := docsCache[cacheKey]; !ok {
		doc, err := discovery.FetchDiscoveryDocument(ctx, discClient, serviceName, version)
		if err != nil {
			log.Printf("Failed to fetch discovery doc for %s: %v", cacheKey, err)
			return
		}
		docsCache[cacheKey] = doc
	}

	doc := docsCache[cacheKey]
	methodName := parts[len(parts)-1]
	resourcePath := parts[2 : len(parts)-1]

	method, err := discovery.FindMethod(doc, resourcePath, methodName)
	if err != nil {
		log.Printf("Failed to find method %s in %s: %v", endpoint, cacheKey, err)
		return
	}

	schema := discovery.MethodToSchema(method, doc)
	mcpTool := &mcp.Tool{
		Name:        endpoint,
		Description: method.Description,
		InputSchema: schema,
	}

	mcp.AddTool(server, mcpTool, func(ctx context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		callResult, err := executeDynamicCall(ctx, clientProvider, doc, method, args)
		return callResult, nil, err
	})
}

func executeDynamicCall(
	ctx context.Context,
	clientProvider ClientProvider,
	doc *discovery.RestDescription,
	method *discovery.RestMethod,
	parsed map[string]any,
) (*mcp.CallToolResult, error) {
	client, err := clientProvider(ctx)
	if err != nil {
		return nil, err
	}

	req, err := discovery.BuildRequest(ctx, doc, method, parsed, true)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("unexpected HTTP %d from API: %s", resp.StatusCode, string(respBody))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSONOrRaw(respBody)},
		},
	}, nil
}

func formatJSONOrRaw(respBody []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBody, "", "  "); err != nil {
		return string(respBody)
	}
	return prettyJSON.String()
}
