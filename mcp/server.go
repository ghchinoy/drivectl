package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/ghchinoy/drivectl/internal/discovery"
	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
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

// getHTTPClient creates an authenticated HTTP client for API requests.
func getHTTPClient(ctx context.Context) (*http.Client, error) {
	viper.AutomaticEnv()
	secretFile := viper.GetString("secret-file")
	if secretFile == "" {
		configDir, err := drive.ConfigDir()
		if err == nil {
			// fallback to default
			secretFile = configDir + "/client_secret.json"
		}
	}
	noBrowserAuth := viper.GetBool("no-browser-auth")
	client, err := drive.NewOAuthClient(ctx, secretFile, noBrowserAuth)
	if err != nil {
		return nil, fmt.Errorf("could not create oauth client: %w", err)
	}
	return client, nil
}

// driveQueryCheatSheetHandler is a resource handler that returns a cheat sheet of Drive query examples.
func driveQueryCheatSheetHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      params.URI,
				MIMEType: "text/plain",
				Text:     driveQueryCheatSheet,
			},
		},
	}, nil
}

// a1NotationCheatSheetHandler is a resource handler that returns a cheat sheet of A1 notation examples.
func a1NotationCheatSheetHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      params.URI,
				MIMEType: "text/plain",
				Text:     a1NotationCheatSheet,
			},
		},
	}, nil
}

// Dynamic tools to expose via MCP. Format: service.version.resource.method
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
}

// registerToolsAndResources registers all dynamic API tools and local resources.
func registerToolsAndResources(server *mcp.Server) error {
	// Add Resources
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

	// Fetch discovery docs for our key services to build exact input schemas
	docsCache := make(map[string]*discovery.RestDescription)

	for _, endpoint := range dynamicTools {
		parts := strings.Split(endpoint, ".")
		serviceName := parts[0]
		version := parts[1]
		cacheKey := serviceName + "_" + version

		if _, ok := docsCache[cacheKey]; !ok {
			doc, err := discovery.FetchDiscoveryDocument(http.DefaultClient, serviceName, version)
			if err != nil {
				log.Printf("Failed to fetch discovery doc for %s: %v", cacheKey, err)
				continue
			}
			docsCache[cacheKey] = doc
		}

		doc := docsCache[cacheKey]
		methodName := parts[len(parts)-1]
		resourcePath := parts[2 : len(parts)-1]
		
		method, err := findMethod(doc, resourcePath, methodName)
		if err != nil {
			log.Printf("Failed to find method %s in %s: %v", endpoint, cacheKey, err)
			continue
		}

		schema := discovery.MethodToSchema(method, doc)

		// Create a local copy of endpoint and method for the closure
		localEndpoint := endpoint
		localMethod := method
		localDoc := doc

		mcpTool := &mcp.Tool{
			Name:        localEndpoint,
			Description: localMethod.Description,
			InputSchema: schema,
		}

		mcp.AddTool(server, mcpTool, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]interface{}]) (*mcp.CallToolResultFor[any], error) {
			return executeDynamicCall(ctx, localEndpoint, localDoc, localMethod, params.Arguments)
		})
		log.Printf("Registered MCP tool: %s", localEndpoint)
	}

	return nil
}

func executeDynamicCall(ctx context.Context, endpoint string, doc *discovery.RestDescription, method *discovery.RestMethod, parsed map[string]interface{}) (*mcp.CallToolResultFor[any], error) {
	client, err := getHTTPClient(ctx)
	if err != nil {
		return nil, err
	}

	apiURL := doc.BaseURL + method.Path
	if doc.BaseURL == "" {
		apiURL = doc.RootURL + doc.ServicePath + method.Path
	}

	var reqBody io.Reader
	if parsed != nil {
		// Extract path parameters
		for paramName, paramDef := range method.Parameters {
			if paramDef.Location == "path" {
				if val, ok := parsed[paramName]; ok {
					valStr := fmt.Sprintf("%v", val)
					apiURL = strings.Replace(apiURL, "{"+paramName+"}", url.PathEscape(valStr), -1)
					delete(parsed, paramName)
				} else if paramDef.Required {
					return nil, fmt.Errorf("missing required path parameter: %s", paramName)
				}
			}
		}

		if method.Request == nil {
			// Add remaining payload as query parameters
			u, err := url.Parse(apiURL)
			if err == nil {
				q := u.Query()
				for k, v := range parsed {
					q.Set(k, fmt.Sprintf("%v", v))
				}
				u.RawQuery = q.Encode()
				apiURL = u.String()
			}
		} else {
			// Extract payload body if we nested it under "payload" to separate it from path params
			var bodyData interface{} = parsed
			if payloadObj, ok := parsed["payload"]; ok {
				bodyData = payloadObj
			}
			jsonBytes, _ := json.Marshal(bodyData)
			reqBody = bytes.NewReader(jsonBytes)
		}
	} else {
		for paramName, paramDef := range method.Parameters {
			if paramDef.Location == "path" && paramDef.Required {
				return nil, fmt.Errorf("missing required path parameter: %s", paramName)
			}
		}
	}

	req, err := http.NewRequest(method.HTTPMethod, apiURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add the auth header explicitly using the client context (which handles token refresh)
	driveSvc, err := googledrive.NewService(ctx, option.WithHTTPClient(client))
	if err == nil {
		// Just a dummy call to ensure token is valid/refreshed, wait, client.Do() handles it
	}
	_ = driveSvc

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	// Format output
	var prettyJSON bytes.Buffer
	var output string
	if err := json.Indent(&prettyJSON, respBody, "", "  "); err != nil {
		output = string(respBody)
	} else {
		output = prettyJSON.String()
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil
}

func findMethod(doc *discovery.RestDescription, resourcePath []string, methodName string) (*discovery.RestMethod, error) {
	if len(resourcePath) == 0 {
		return nil, fmt.Errorf("resource path cannot be empty")
	}

	firstResourceName := resourcePath[0]
	resource, ok := doc.Resources[firstResourceName]
	if !ok {
		return nil, fmt.Errorf("resource '%s' not found", firstResourceName)
	}

	currentResource := resource
	for _, subName := range resourcePath[1:] {
		subResource, ok := currentResource.Resources[subName]
		if !ok {
			return nil, fmt.Errorf("sub-resource '%s' not found", subName)
		}
		currentResource = subResource
	}

	method, ok := currentResource.Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method '%s' not found", methodName)
	}

	return &method, nil
}

// Start starts the MCP server.
func Start(rootCmd *cobra.Command, httpAddr string) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "drivectl"}, nil)

	if err := registerToolsAndResources(server); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	if httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", httpAddr)
		return http.ListenAndServe(httpAddr, handler)
	}

	logFile, err := os.OpenFile("drivectl-mcp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), logFile)
	if err := server.Run(context.Background(), t); err != nil {
		log.Printf("Server failed: %v", err)
		return err
	}

	return nil
}