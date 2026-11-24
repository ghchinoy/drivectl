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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/discovery"
)

var (
	payloadData string
	payloadFile string
)

var callCmd = &cobra.Command{
	Use:     "call [service.resource.method]",
	GroupID: GroupAdvanced,
	Short:   "Call an arbitrary Google API endpoint dynamically",
	Long: `Fetches the Google API Discovery Document for the specified service, validates
the provided JSON payload (if any), and executes the API call dynamically.
Use --payload for inline JSON or --payload-file <path> (or '-' for stdin) for larger bodies.

Example:
  drivectl call drive.v3.files.list
  drivectl call docs.v1.documents.create --payload '{"title": "My New Doc"}'
  drivectl call slides.v1.presentations.batchUpdate --payload-file ./requests.json
  cat ./requests.json | drivectl call slides.v1.presentations.batchUpdate --payload-file -
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		parts := strings.Split(path, ".")
		if len(parts) < 4 {
			return ui.ErrorWithHint(
				errors.New("invalid path format"),
				"Path must be in the format 'service.version.resource.method' (e.g. drive.v3.files.list)",
			)
		}

		serviceName := parts[0]
		version := parts[1]
		methodName := parts[len(parts)-1]
		resourcePath := parts[2 : len(parts)-1]

		ctx := cmd.Context()
		httpClient := client
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		doc, err := discovery.FetchDiscoveryDocument(ctx, httpClient, serviceName, version)
		if err != nil {
			wrapped := fmt.Errorf("failed to fetch discovery document: %w", err)
			return ui.ErrorWithHint(wrapped, "Ensure your network is active and the API service/version exist.")
		}

		method, err := discovery.FindMethod(doc, resourcePath, methodName)
		if err != nil {
			return ui.ErrorWithHint(err, "Double check the resource and method names against the Google API documentation.")
		}

		parsed, err := loadCallPayload(cmd, payloadData, payloadFile)
		if err != nil {
			return ui.ErrorWithHint(err, "Provide valid JSON via either --payload or --payload-file (use '-' for stdin).")
		}

		req, err := discovery.BuildRequest(ctx, doc, method, parsed, false)
		if err != nil {
			return err
		}

		if dryRun {
			return previewHTTPRequest("call "+path, req)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			wrapped := fmt.Errorf("executing API request: %w", err)
			return ui.ErrorWithHint(wrapped, "Check your network connection and payload formatting.")
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			wrapped := fmt.Errorf("failed to read response: %w", err)
			return ui.ErrorWithHint(wrapped, "The server returned an unreadable response.")
		}

		if resp.StatusCode >= http.StatusBadRequest {
			wrapped := fmt.Errorf("unexpected HTTP %d from API: %s", resp.StatusCode, string(respBody))
			return ui.ErrorWithHint(wrapped, "Read the Google API error message above to correct your request.")
		}

		fmt.Println(prettyFormatResponse(respBody))
		return nil
	},
}

func loadCallPayload(cmd *cobra.Command, inline, filePath string) (map[string]any, error) {
	if inline != "" && filePath != "" {
		return nil, errors.New("cannot specify both --payload and --payload-file")
	}

	raw, err := readCallPayloadBytes(cmd, inline, filePath)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	return parsed, nil
}

func readCallPayloadBytes(cmd *cobra.Command, inline, filePath string) ([]byte, error) {
	switch {
	case filePath == "-":
		raw, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("reading payload from stdin: %w", err)
		}
		return raw, nil
	case filePath != "":
		raw, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading payload file %q: %w", filePath, err)
		}
		return raw, nil
	default:
		return []byte(inline), nil
	}
}

func prettyFormatResponse(respBody []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBody, "", "  "); err != nil {
		return string(respBody)
	}
	return prettyJSON.String()
}

func init() {
	rootCmd.AddCommand(callCmd)
	callCmd.Flags().StringVar(&payloadData, "payload", "", "JSON payload for the request (includes path parameters)")
	callCmd.Flags().StringVar(&payloadFile, "payload-file", "", "Path to a JSON file for the request payload (use '-' for stdin)")
}
