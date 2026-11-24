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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDryRun_DoesNotExecuteWrites(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := t.Context()

	tmpDir := t.TempDir()
	localDeck := filepath.Join(tmpDir, "deck.pptx")
	if err := os.WriteFile(localDeck, []byte("fake pptx bytes"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", localDeck, err)
	}
	localMD := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(localMD, []byte("# Title\n\nHello **world**."), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", localMD, err)
	}

	writeAttempts := 0
	origClient := client
	client = &http.Client{
		Transport: cmdRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/discovery/v1/apis/drive/v3/rest") {
				return cmdJSONResp(req, http.StatusOK, `{
					"name":"drive",
					"version":"v3",
					"baseUrl":"https://www.googleapis.com/drive/v3/",
					"schemas":{"File":{"id":"File","type":"object","properties":{"name":{"type":"string"}}}},
					"resources":{"files":{"methods":{"create":{"id":"drive.files.create","httpMethod":"POST","path":"files","request":{"$ref":"File"}}}}}
				}`), nil
			}
			writeAttempts++
			return cmdJSONResp(req, http.StatusInternalServerError, `{"error":"write should not be executed during dry-run"}`), nil
		}),
	}
	t.Cleanup(func() {
		client = origClient
		dryRun = false
		OutputFormat = ""
		uploadFileID = ""
		uploadConvert = ""
		uploadName = ""
		parentID = ""
		payloadData = ""
		payloadFile = ""
		slidesNotesSlide = 0
		slidesNotesSet = ""
		slidesNotesFile = ""
		if f := slidesNotesCmd.Flags().Lookup("set"); f != nil {
			f.Changed = false
		}
		if f := slidesNotesCmd.Flags().Lookup("file"); f != nil {
			f.Changed = false
		}
	})

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "call dry-run json",
			args: []string{"call", "drive.v3.files.create", "--payload", `{"name":"Draft"}`, "--dry-run", "-O", "json"},
		},
		{
			name: "upload create convert dry-run",
			args: []string{"upload", localDeck, "--convert", "slides", "--dry-run", "-O", "json"},
		},
		{
			name: "upload update in-place dry-run",
			args: []string{"upload", localDeck, "--file-id", "pres-123", "--convert", "slides", "--dry-run"},
		},
		{
			name: "docs create dry-run",
			args: []string{"docs", "create", "Design Spec", localMD, "--dry-run", "-O", "json"},
		},
		{
			name: "sheets update-range dry-run",
			args: []string{"sheets", "update-range", "sheet-1", "Done", "--sheet", "Sheet1", "--range", "B2", "--dry-run", "-O", "json"},
		},
		{
			name: "slides create dry-run",
			args: []string{"slides", "create", "Workshop Deck", "--dry-run", "-O", "json"},
		},
		{
			name: "slides replace dry-run",
			args: []string{"slides", "replace", "pres-1", "Old", "New", "--match-case", "--dry-run", "-O", "json"},
		},
		{
			name: "slides notes set dry-run",
			args: []string{"slides", "notes", "pres-1", "--slide", "1", "--set", "Speaker cue", "--dry-run", "-O", "json"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dryRun = false
			OutputFormat = ""
			uploadFileID = ""
			uploadConvert = ""
			uploadName = ""
			parentID = ""
			payloadData = ""
			payloadFile = ""
			slidesNotesSlide = 0
			slidesNotesSet = ""
			slidesNotesFile = ""
			if f := slidesNotesCmd.Flags().Lookup("set"); f != nil {
				f.Changed = false
			}
			if f := slidesNotesCmd.Flags().Lookup("file"); f != nil {
				f.Changed = false
			}

			rootCmd.SetArgs(tc.args)
			if err := ExecuteContext(ctx); err != nil {
				t.Fatalf("ExecuteContext(%v) error = %v", tc.args, err)
			}
		})
	}

	if writeAttempts != 0 {
		t.Fatalf("dry-run triggered %d HTTP write request(s), want 0", writeAttempts)
	}
}

func TestDryRun_WorksWithoutOAuthCredentials(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DRIVECTL_TOKEN", "")
	oldClient := client
	client = nil
	t.Cleanup(func() {
		client = oldClient
		dryRun = false
		OutputFormat = ""
	})

	rootCmd.SetArgs([]string{"slides", "create", "Offline Preview", "--dry-run", "-O", "json"})
	if err := ExecuteContext(t.Context()); err != nil {
		t.Fatalf("ExecuteContext(slides create --dry-run) without OAuth credentials error = %v", err)
	}
}
