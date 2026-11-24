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

package drive

import (
	"fmt"
	"slices"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"
)

// AccessTokenEnvVar is the environment variable for supplying a pre-obtained OAuth 2.0 access token.
const AccessTokenEnvVar = "DRIVECTL_TOKEN"

// ReadOnlyOAuthScopes returns the read-only Google Workspace OAuth scopes across all supported services.
func ReadOnlyOAuthScopes() []string {
	return []string{
		drive.DriveReadonlyScope,
		docs.DocumentsReadonlyScope,
		sheets.SpreadsheetsReadonlyScope,
		slides.PresentationsReadonlyScope,
	}
}

// ResolveScopes resolves Google Workspace OAuth scopes for the requested services and access mode.
// When services is empty, all supported services (drive, docs, sheets, slides) are included.
func ResolveScopes(services []string, readOnly bool) ([]string, error) {
	normalized, err := normalizeServices(services)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		if readOnly {
			return ReadOnlyOAuthScopes(), nil
		}
		return DefaultOAuthScopes(), nil
	}

	var resolved []string
	for _, svc := range normalized {
		resolved = append(resolved, scopesForService(svc, readOnly)...)
	}
	return deduplicateScopes(resolved), nil
}

// MergeScopes returns the deduplicated union of existing and requested scopes, preserving order.
func MergeScopes(existing, requested []string) []string {
	combined := make([]string, 0, len(existing)+len(requested))
	combined = append(combined, existing...)
	combined = append(combined, requested...)
	return deduplicateScopes(combined)
}

// ComputeLoginScopes calculates the scopes to request during an interactive login.
// When services is non-empty and replaceScopes is false, newly requested scopes are merged
// incrementally with existing profile scopes (filtering to read-only scopes when readOnly is set).
func ComputeLoginScopes(existing, services []string, readOnly, replaceScopes bool) ([]string, error) {
	requested, err := ResolveScopes(services, readOnly)
	if err != nil {
		return nil, err
	}
	if replaceScopes || len(existing) == 0 || len(services) == 0 {
		return requested, nil
	}
	if readOnly {
		return MergeScopes(filterReadOnlyScopes(existing), requested), nil
	}
	return MergeScopes(existing, requested), nil
}

func normalizeServices(raw []string) ([]string, error) {
	var out []string
	seen := make(map[string]bool)
	for _, entry := range raw {
		for _, part := range strings.Split(entry, ",") {
			svc := strings.ToLower(strings.TrimSpace(part))
			if svc == "" {
				continue
			}
			if !isValidService(svc) {
				return nil, fmt.Errorf("unknown service %q (valid services: drive, docs, sheets, slides)", svc)
			}
			if !seen[svc] {
				seen[svc] = true
				out = append(out, svc)
			}
		}
	}
	return out, nil
}

func isValidService(svc string) bool {
	switch svc {
	case "drive", "docs", "sheets", "slides":
		return true
	default:
		return false
	}
}

func scopesForService(svc string, readOnly bool) []string {
	switch svc {
	case "drive":
		if readOnly {
			return []string{drive.DriveReadonlyScope}
		}
		return []string{drive.DriveReadonlyScope, drive.DriveFileScope}
	case "docs":
		if readOnly {
			return []string{docs.DocumentsReadonlyScope}
		}
		return []string{docs.DocumentsReadonlyScope, drive.DriveFileScope}
	case "sheets":
		if readOnly {
			return []string{sheets.SpreadsheetsReadonlyScope}
		}
		return []string{sheets.SpreadsheetsScope}
	case "slides":
		if readOnly {
			return []string{slides.PresentationsReadonlyScope}
		}
		return []string{slides.PresentationsScope}
	default:
		return nil
	}
}

func filterReadOnlyScopes(scopes []string) []string {
	var out []string
	for _, sc := range scopes {
		if strings.HasSuffix(sc, ".readonly") {
			out = append(out, sc)
		}
	}
	return out
}

func deduplicateScopes(scopes []string) []string {
	var out []string
	for _, sc := range scopes {
		trimmed := strings.TrimSpace(sc)
		if trimmed == "" || slices.Contains(out, trimmed) {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
