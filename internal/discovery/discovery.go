package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// RestDescription represents the top-level Discovery Document.
type RestDescription struct {
	Name        string                       `json:"name"`
	Version     string                       `json:"version"`
	Title       string                       `json:"title"`
	Description string                       `json:"description"`
	RootURL     string                       `json:"rootUrl"`
	ServicePath string                       `json:"servicePath"`
	BaseURL     string                       `json:"baseUrl"`
	Schemas     map[string]JsonSchema        `json:"schemas"`
	Resources   map[string]RestResource      `json:"resources"`
	Parameters  map[string]MethodParameter   `json:"parameters"`
}

// RestResource represents a resource in the API.
type RestResource struct {
	Methods   map[string]RestMethod   `json:"methods"`
	Resources map[string]RestResource `json:"resources"`
}

// RestMethod represents a method in the API.
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

// SchemaRef represents a reference to a schema.
type SchemaRef struct {
	Ref string `json:"$ref"`
}

// MethodParameter represents a parameter for a method.
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

// JsonSchema represents a schema definition.
type JsonSchema struct {
	ID                   string                        `json:"id"`
	Type                 string                        `json:"type"`
	Description          string                        `json:"description"`
	Properties           map[string]JsonSchemaProperty `json:"properties"`
	Ref                  string                        `json:"$ref"`
	Items                *JsonSchemaProperty           `json:"items"`
	Required             []string                      `json:"required"`
	AdditionalProperties *JsonSchemaProperty           `json:"additionalProperties"`
}

// JsonSchemaProperty represents a property within a schema.
type JsonSchemaProperty struct {
	Type                 string                        `json:"type"`
	Description          string                        `json:"description"`
	Ref                  string                        `json:"$ref"`
	Format               string                        `json:"format"`
	Items                *JsonSchemaProperty           `json:"items"`
	Properties           map[string]JsonSchemaProperty `json:"properties"`
	ReadOnly             bool                          `json:"readOnly"`
	Default              string                        `json:"default"`
	Enum                 []string                      `json:"enum"`
	AdditionalProperties *JsonSchemaProperty           `json:"additionalProperties"`
}

// configDir returns the path to the application's config directory.
func configDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	dir := filepath.Join(homeDir, ".config", "drivectl")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return dir, nil
}

// FetchDiscoveryDocument fetches and caches a Google Discovery Document.
func FetchDiscoveryDocument(client *http.Client, service string, version string) (*RestDescription, error) {
	cacheDir, err := configDir()
	if err != nil {
		return nil, err
	}
	
	cacheSubDir := filepath.Join(cacheDir, "discovery")
	if err := os.MkdirAll(cacheSubDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create discovery cache directory: %w", err)
	}

	cacheFile := filepath.Join(cacheSubDir, fmt.Sprintf("%s_%s.json", service, version))

	// Check cache (24hr TTL)
	if stat, err := os.Stat(cacheFile); err == nil {
		if time.Since(stat.ModTime()) < 24*time.Hour {
			data, err := os.ReadFile(cacheFile)
			if err == nil {
				var doc RestDescription
				if err := json.Unmarshal(data, &doc); err == nil {
					return &doc, nil
				}
			}
		}
	}

	// Fetch from network
	url := fmt.Sprintf("https://www.googleapis.com/discovery/v1/apis/%s/%s/rest", service, version)
	
	// Fallback to $discovery/rest for some APIs (like Forms/Keep)
	if service == "forms" || service == "keep" || service == "meet" || service == "chat" {
		url = fmt.Sprintf("https://%s.googleapis.com/$discovery/rest?version=%s", service, version)
	}

	req, err := http.NewRequest("GET", url, nil)
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

	// Write to cache
	if err := os.WriteFile(cacheFile, body, 0600); err != nil {
		// Non-fatal, just log/ignore
		fmt.Fprintf(os.Stderr, "Warning: failed to cache discovery document: %v\n", err)
	}

	var doc RestDescription
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	return &doc, nil
}
