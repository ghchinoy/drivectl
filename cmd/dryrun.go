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
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/ghchinoy/drivectl/internal/ui"
)

type dryRunPlan struct {
	DryRun  bool           `json:"dryRun"`
	Command string         `json:"command"`
	Method  string         `json:"method"`
	URL     string         `json:"url"`
	Headers map[string]any `json:"headers,omitempty"`
	Payload any            `json:"payload,omitempty"`
	Notes   []string       `json:"notes,omitempty"`
}

func isDryRunOfflineCommand(cmd *cobra.Command) bool {
	if !dryRun {
		return false
	}
	switch commandPathKey(cmd) {
	case "call", "upload", "docs create", "sheets update-range", "slides create", "slides replace":
		return true
	case "slides notes":
		return cmd.Flags().Changed("set") || cmd.Flags().Changed("file")
	default:
		return false
	}
}

func printDryRunPlan(plan dryRunPlan) error {
	plan.DryRun = true
	if OutputFormat == "json" {
		return printJSON(plan)
	}

	fmt.Printf("%s %s %s\n", ui.Accent("[DRY RUN]"), plan.Method, plan.URL)
	if len(plan.Headers) > 0 {
		for k, v := range plan.Headers {
			fmt.Printf("  Header: %s: %v\n", k, v)
		}
	}
	if plan.Payload != nil {
		b, err := json.MarshalIndent(plan.Payload, "  ", "  ")
		if err == nil {
			fmt.Printf("  Payload:\n  %s\n", string(b))
		}
	}
	for _, note := range plan.Notes {
		fmt.Printf("  Note: %s\n", ui.Muted(note))
	}
	return nil
}

func previewHTTPRequest(command string, req *http.Request) error {
	headers := make(map[string]any)
	if ct := req.Header.Get("Content-Type"); ct != "" {
		headers["Content-Type"] = ct
	}

	var payload any
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("reading request body for dry-run preview: %w", err)
		}
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) > 0 {
			payload = decodeJSONOrRaw(trimmed)
		}
	}

	return printDryRunPlan(dryRunPlan{
		Command: command,
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: headers,
		Payload: payload,
	})
}

func decodeJSONOrRaw(raw []byte) any {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		return decoded
	}
	return string(raw)
}
