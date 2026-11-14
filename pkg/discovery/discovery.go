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

// Package discovery fetches, caches, and converts Google API Discovery Documents.
package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cacheTTL = 24 * time.Hour

// RestDescription represents the top-level Google API Discovery Document.
type RestDescription struct {
	Name        string                     `json:"name"`
	Version     string                     `json:"version"`
	Title       string                     `json:"title"`
	Description string                     `json:"description"`
	RootURL     string                     `json:"rootUrl"`
	ServicePath string                     `json:"servicePath"`
	BaseURL     string                     `json:"baseUrl"`
	Schemas     map[string]JSONSchema      `json:"schemas"`
	Resources   map[string]RestResource    `json:"resources"`
	Parameters  map[string]MethodParameter `json:"parameters"`
}

// RestResource represents a resource in a Google API Discovery Document.
type RestResource struct {
	Methods   map[string]RestMethod   `json:"methods"`
	Resources map[string]RestResource `json:"resources"`
}

// RestMethod represents a method on a resource in a Google API Discovery Document.
type RestMethod struct {
	ID                  string                     `json:"id"`
	Description         string                     `json:"description"`
	HTTPMethod          string                     `json:"httpMethod"`
	Path                string                     `json:"path"`
	Parameters          map[string]MethodParameter `json:"parameters"`
	ParameterOrder      []string                   `json:"parameterOrder"`
	Request             *SchemaRef                 `json:"request"`
	Response            *SchemaRef                 `json:"response"`
	Scopes              []string                   `json:"scopes"`
	SupportsMediaUpload bool                       `json:"supportsMediaUpload"`
}

// SchemaRef represents a reference to a named schema in a Discovery Document.
type SchemaRef struct {
	Ref string `json:"$ref"`
}

// MethodParameter represents a parameter for a Discovery method.
type MethodParameter struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Location    string   `json:"location"`
	Required    bool     `json:"required"`
	Format      string   `json:"format"`
	Default     string   `json:"default"`
	Enum        []string `json:"enum"`
	Repeated    bool     `json:"repeated"`
}

// JSONSchema represents a schema definition in a Discovery Document.
type JSONSchema struct {
	ID                   string                        `json:"id"`
	Type                 string                        `json:"type"`
	Description          string                        `json:"description"`
	Properties           map[string]JSONSchemaProperty `json:"properties"`
	Ref                  string                        `json:"$ref"`
	Items                *JSONSchemaProperty           `json:"items"`
	Required             []string                      `json:"required"`
	AdditionalProperties *JSONSchemaProperty           `json:"additionalProperties"`
}

// JSONSchemaProperty represents a property within a Discovery JSONSchema.
type JSONSchemaProperty struct {
	Type                 string                        `json:"type"`
	Description          string                        `json:"description"`
	Ref                  string                        `json:"$ref"`
	Format               string                        `json:"format"`
	Items                *JSONSchemaProperty           `json:"items"`
	Properties           map[string]JSONSchemaProperty `json:"properties"`
	ReadOnly             bool                          `json:"readOnly"`
	Default              string                        `json:"default"`
	Enum                 []string                      `json:"enum"`
	AdditionalProperties *JSONSchemaProperty           `json:"additionalProperties"`
}

// FetchDiscoveryDocument fetches and caches a Google API Discovery Document.
func FetchDiscoveryDocument(ctx context.Context, client *http.Client, service, version string) (*RestDescription, error) {
	cacheFile, err := discoveryCacheFile(service, version)
	if err != nil {
		return nil, err
	}

	if doc, ok := readCachedDiscoveryDoc(cacheFile); ok {
		return doc, nil
	}

	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL(service, version), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch discovery document: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := os.WriteFile(cacheFile, body, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache discovery document: %v\n", err)
	}

	var doc RestDescription
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	return &doc, nil
}

// CallMethod resolves a dot-separated endpoint (such as "drive.v3.files.get")
// against its Discovery Document, executes the HTTP request, and returns the raw response body.
func CallMethod(ctx context.Context, client *http.Client, endpoint string, payload map[string]any) ([]byte, error) {
	parts := strings.Split(endpoint, ".")
	if len(parts) < 4 {
		return nil, errors.New("invalid endpoint format: expected 'service.version.resource.method'")
	}

	serviceName := parts[0]
	version := parts[1]
	methodName := parts[len(parts)-1]
	resourcePath := parts[2 : len(parts)-1]

	doc, err := FetchDiscoveryDocument(ctx, client, serviceName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}

	return doc.CallMethod(ctx, client, resourcePath, methodName, payload, false)
}

// CallMethod resolves a nested resource path and method name inside d,
// builds and executes the HTTP request, and returns the raw response body.
func (d *RestDescription) CallMethod(ctx context.Context, client *http.Client, resourcePath []string, methodName string, payload map[string]any, unwrapPayloadKey bool) ([]byte, error) {
	method, err := FindMethod(d, resourcePath, methodName)
	if err != nil {
		return nil, err
	}

	req, err := BuildRequest(ctx, d, method, payload, unwrapPayloadKey)
	if err != nil {
		return nil, err
	}

	if client == nil {
		client = http.DefaultClient
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

	return respBody, nil
}

// FindMethod resolves a nested resource path and method name inside a Discovery Document.
func FindMethod(doc *RestDescription, resourcePath []string, methodName string) (*RestMethod, error) {
	if doc == nil {
		return nil, errors.New("discovery document cannot be nil")
	}
	if len(resourcePath) == 0 {
		return nil, errors.New("resource path cannot be empty")
	}

	firstResourceName := resourcePath[0]
	currentResource, ok := doc.Resources[firstResourceName]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", firstResourceName)
	}

	for _, subName := range resourcePath[1:] {
		subResource, ok := currentResource.Resources[subName]
		if !ok {
			return nil, fmt.Errorf("sub-resource %q not found", subName)
		}
		currentResource = subResource
	}

	method, ok := currentResource.Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method %q not found", methodName)
	}

	return &method, nil
}

// BuildRequest constructs an HTTP request for a Discovery method from a parsed argument map.
func BuildRequest(ctx context.Context, doc *RestDescription, method *RestMethod, parsed map[string]any, unwrapPayloadKey bool) (*http.Request, error) {
	apiURL := resolveMethodURL(doc, method)
	params := maps.Clone(parsed)

	var err error
	apiURL, err = substitutePathParams(apiURL, method, params)
	if err != nil {
		return nil, err
	}

	apiURL, err = extractQueryParams(apiURL, doc, method, params)
	if err != nil {
		return nil, err
	}

	reqBody, apiURL, err := prepareQueryOrBody(apiURL, method, params, unwrapPayloadKey)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method.HTTPMethod, apiURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func resolveMethodURL(doc *RestDescription, method *RestMethod) string {
	if doc != nil && doc.BaseURL != "" {
		return doc.BaseURL + method.Path
	}
	if doc != nil {
		return doc.RootURL + doc.ServicePath + method.Path
	}
	return method.Path
}

func substitutePathParams(apiURL string, method *RestMethod, params map[string]any) (string, error) {
	for paramName, paramDef := range method.Parameters {
		if paramDef.Location != "path" {
			continue
		}
		if val, ok := params[paramName]; ok {
			valStr := fmt.Sprint(val)
			apiURL = strings.ReplaceAll(apiURL, "{"+paramName+"}", url.PathEscape(valStr))
			delete(params, paramName)
			continue
		}
		if paramDef.Required {
			return "", fmt.Errorf("missing required path parameter %q", paramName)
		}
	}
	return apiURL, nil
}

func extractQueryParams(apiURL string, doc *RestDescription, method *RestMethod, params map[string]any) (string, error) {
	if len(params) == 0 {
		return apiURL, nil
	}
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("invalid API URL %q: %w", apiURL, err)
	}
	q := u.Query()
	modified := false
	for k, v := range params {
		if isQueryParameter(doc, method, k) {
			q.Set(k, fmt.Sprint(v))
			delete(params, k)
			modified = true
		}
	}
	if !modified {
		return apiURL, nil
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func isQueryParameter(doc *RestDescription, method *RestMethod, name string) bool {
	if method != nil {
		if paramDef, ok := method.Parameters[name]; ok && paramDef.Location == "query" {
			return true
		}
	}
	if doc != nil {
		if paramDef, ok := doc.Parameters[name]; ok && paramDef.Location == "query" {
			return true
		}
	}
	return false
}

func prepareQueryOrBody(apiURL string, method *RestMethod, params map[string]any, unwrapPayloadKey bool) (io.Reader, string, error) {
	if len(params) == 0 && method.Request == nil {
		return nil, apiURL, nil
	}
	if method.Request == nil {
		u, err := url.Parse(apiURL)
		if err != nil {
			return nil, "", fmt.Errorf("invalid API URL %q: %w", apiURL, err)
		}
		q := u.Query()
		for k, v := range params {
			q.Set(k, fmt.Sprint(v))
		}
		u.RawQuery = q.Encode()
		return nil, u.String(), nil
	}
	if params == nil {
		return nil, apiURL, nil
	}
	var bodyData any = params
	if unwrapPayloadKey {
		if payloadObj, ok := params["payload"]; ok {
			bodyData = payloadObj
		}
	}
	jsonBytes, err := json.Marshal(bodyData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	return bytes.NewReader(jsonBytes), apiURL, nil
}

func configDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	dir := filepath.Join(homeDir, ".config", "drivectl")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return dir, nil
}

func discoveryCacheFile(service, version string) (string, error) {
	cacheDir, err := configDir()
	if err != nil {
		return "", err
	}
	cacheSubDir := filepath.Join(cacheDir, "discovery")
	if err := os.MkdirAll(cacheSubDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create discovery cache directory: %w", err)
	}
	return filepath.Join(cacheSubDir, fmt.Sprintf("%s_%s.json", service, version)), nil
}

func readCachedDiscoveryDoc(cacheFile string) (*RestDescription, bool) {
	stat, err := os.Stat(cacheFile)
	if err != nil || time.Since(stat.ModTime()) >= cacheTTL {
		return nil, false
	}
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}
	var doc RestDescription
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, false
	}
	return &doc, true
}

func discoveryURL(service, version string) string {
	switch service {
	case "forms", "keep", "meet", "chat":
		return fmt.Sprintf("https://%s.googleapis.com/$discovery/rest?version=%s", service, version)
	default:
		return fmt.Sprintf("https://www.googleapis.com/discovery/v1/apis/%s/%s/rest", service, version)
	}
}
