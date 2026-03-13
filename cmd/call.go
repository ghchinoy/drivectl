package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ghchinoy/drivectl/internal/discovery"
	"github.com/spf13/cobra"
)

var (
	payloadData string
)

var callCmd = &cobra.Command{
	Use:   "call [service.resource.method]",
	Short: "Call an arbitrary Google API endpoint dynamically",
	Long: `Fetches the Google API Discovery Document for the specified service, validates
the provided JSON payload (if any), and executes the API call dynamically.

Example:
  drivectl call drive.v3.files.list
  drivectl call docs.v1.documents.create --payload '{"title": "My New Doc"}'
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		parts := strings.Split(path, ".")
		if len(parts) < 4 {
			return fmt.Errorf("path must be in the format 'service.version.resource.method' (e.g. drive.v3.files.list)")
		}

		serviceName := parts[0]
		version := parts[1]
		methodName := parts[len(parts)-1]
		resourcePath := parts[2 : len(parts)-1]

		doc, err := discovery.FetchDiscoveryDocument(client, serviceName, version)
		if err != nil {
			return fmt.Errorf("failed to fetch discovery document: %w", err)
		}

		method, err := findMethod(doc, resourcePath, methodName)
		if err != nil {
			return err
		}

		// Prepare the URL
		// Simple replacement for path parameters (very naive for now)
		apiURL := doc.BaseURL + method.Path
		// If BaseURL is missing, fallback to rootUrl + servicePath
		if doc.BaseURL == "" {
			apiURL = doc.RootURL + doc.ServicePath + method.Path
		}
		
		var reqBody io.Reader
		if payloadData != "" {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(payloadData), &parsed); err != nil {
				return fmt.Errorf("invalid JSON payload: %w", err)
			}
			
			// If method requires path parameters, extract them from payload and remove from body
			for paramName, paramDef := range method.Parameters {
				if paramDef.Location == "path" {
					if val, ok := parsed[paramName]; ok {
						valStr := fmt.Sprintf("%v", val)
						apiURL = strings.Replace(apiURL, "{"+paramName+"}", url.PathEscape(valStr), -1)
						delete(parsed, paramName)
					} else if paramDef.Required {
						return fmt.Errorf("missing required path parameter: %s", paramName)
					}
				}
			}

			// Add remaining payload as query parameters if no request body is expected
			if method.Request == nil {
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
				// We have a request body schema
				jsonBytes, _ := json.Marshal(parsed)
				reqBody = bytes.NewReader(jsonBytes)
			}
		} else {
			// Check if any path parameters are strictly required but missing
			for paramName, paramDef := range method.Parameters {
				if paramDef.Location == "path" && paramDef.Required {
					return fmt.Errorf("missing required path parameter: %s (provide via --payload)", paramName)
				}
			}
		}

		req, err := http.NewRequest(method.HTTPMethod, apiURL, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		if reqBody != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("API request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
		}

		// Pretty print JSON response
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, respBody, "", "  "); err != nil {
			// Not JSON, just print raw
			fmt.Println(string(respBody))
		} else {
			fmt.Println(prettyJSON.String())
		}

		return nil
	},
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

func init() {
	rootCmd.AddCommand(callCmd)
	callCmd.Flags().StringVar(&payloadData, "payload", "", "JSON payload for the request (includes path parameters)")
}
