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
	"strings"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/ghchinoy/drivectl/pkg/discovery"
	"github.com/spf13/cobra"
)

var payloadData string

var callCmd = &cobra.Command{
	Use:     "call [service.resource.method]",
	GroupID: GroupAdvanced,
	Short:   "Call an arbitrary Google API endpoint dynamically",
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
		doc, err := discovery.FetchDiscoveryDocument(ctx, client, serviceName, version)
		if err != nil {
			wrapped := fmt.Errorf("failed to fetch discovery document: %w", err)
			return ui.ErrorWithHint(wrapped, "Ensure your network is active and the API service/version exist.")
		}

		method, err := discovery.FindMethod(doc, resourcePath, methodName)
		if err != nil {
			return ui.ErrorWithHint(err, "Double check the resource and method names against the Google API documentation.")
		}

		var parsed map[string]any
		if payloadData != "" {
			if err := json.Unmarshal([]byte(payloadData), &parsed); err != nil {
				return fmt.Errorf("invalid JSON payload: %w", err)
			}
		}

		req, err := discovery.BuildRequest(ctx, doc, method, parsed, false)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
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
}
