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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/api/docs/v1"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"

	"github.com/ghchinoy/drivectl/pkg/drive"
)

var commandScopesMap = map[string][]string{
	"list":                {googledrive.DriveReadonlyScope},
	"describe":            {googledrive.DriveReadonlyScope},
	"get":                 {googledrive.DriveReadonlyScope, docs.DocumentsReadonlyScope},
	"revisions":           {googledrive.DriveReadonlyScope},
	"comments":            {googledrive.DriveReadonlyScope},
	"upload":              {googledrive.DriveFileScope},
	"docs":                {docs.DocumentsReadonlyScope, googledrive.DriveFileScope},
	"docs create":         {googledrive.DriveFileScope},
	"docs tabs":           {docs.DocumentsReadonlyScope},
	"sheets":              {sheets.SpreadsheetsScope},
	"sheets get":          {sheets.SpreadsheetsReadonlyScope},
	"sheets get-range":    {sheets.SpreadsheetsReadonlyScope},
	"sheets update-range": {sheets.SpreadsheetsScope},
	"slides":              {slides.PresentationsScope},
	"slides get":          {slides.PresentationsReadonlyScope},
	"slides inspect":      {slides.PresentationsReadonlyScope},
	"slides create":       {slides.PresentationsScope},
	"slides replace":      {slides.PresentationsScope},
	"slides notes":        {slides.PresentationsScope},
	"call":                drive.DefaultOAuthScopes(),
	"run":                 drive.DefaultOAuthScopes(),
}

func commandPathKey(cmd *cobra.Command) string {
	path := strings.TrimSpace(cmd.CommandPath())
	trimmed := strings.TrimPrefix(path, "drivectl")
	return strings.TrimSpace(trimmed)
}

func requiredScopesForCommandPath(key string) []string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if scopes, ok := commandScopesMap[normalized]; ok {
		return scopes
	}
	return drive.DefaultOAuthScopes()
}

func printCommandScopes(cmd *cobra.Command) error {
	key := commandPathKey(cmd)
	return printScopeList(key, requiredScopesForCommandPath(key))
}

func printScopeList(target string, scopes []string) error {
	if OutputFormat == "json" {
		return printJSON(map[string]any{
			"command": target,
			"scopes":  scopes,
		})
	}
	if target != "" {
		fmt.Printf("Required OAuth scopes for %q:\n", target)
	}
	for _, sc := range scopes {
		fmt.Println(sc)
	}
	return nil
}
