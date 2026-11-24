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
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
	"google.golang.org/api/docs/v1"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"
)

func TestResolveScopes(t *testing.T) {
	tests := []struct {
		name     string
		services []string
		readOnly bool
		want     []string
		wantErr  bool
	}{
		{
			name:     "default all services read-write",
			services: nil,
			readOnly: false,
			want:     DefaultOAuthScopes(),
		},
		{
			name:     "all services read-only",
			services: nil,
			readOnly: true,
			want:     ReadOnlyOAuthScopes(),
		},
		{
			name:     "drive and docs read-only comma separated",
			services: []string{"drive, docs"},
			readOnly: true,
			want:     []string{googledrive.DriveReadonlyScope, docs.DocumentsReadonlyScope},
		},
		{
			name:     "sheets and slides read-write",
			services: []string{"sheets", "slides"},
			readOnly: false,
			want:     []string{sheets.SpreadsheetsScope, slides.PresentationsScope},
		},
		{
			name:     "drive and docs deduplicates drive.file",
			services: []string{"drive", "docs"},
			readOnly: false,
			want:     []string{googledrive.DriveReadonlyScope, googledrive.DriveFileScope, docs.DocumentsReadonlyScope},
		},
		{
			name:     "unknown service returns error",
			services: []string{"gmail"},
			readOnly: false,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveScopes(tc.services, tc.readOnly)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveScopes(%v, %t) error = %v, wantErr %t", tc.services, tc.readOnly, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ResolveScopes(%v, %t) mismatch (-want +got):\n%s", tc.services, tc.readOnly, diff)
			}
		})
	}
}

func TestComputeLoginScopes(t *testing.T) {
	tests := []struct {
		name          string
		existing      []string
		services      []string
		readOnly      bool
		replaceScopes bool
		want          []string
	}{
		{
			name:     "incremental merge adds sheets to existing drive",
			existing: []string{googledrive.DriveReadonlyScope},
			services: []string{"sheets"},
			readOnly: false,
			want:     []string{googledrive.DriveReadonlyScope, sheets.SpreadsheetsScope},
		},
		{
			name:     "readonly incremental filters out existing write scopes",
			existing: []string{googledrive.DriveReadonlyScope, googledrive.DriveFileScope, sheets.SpreadsheetsScope},
			services: []string{"docs"},
			readOnly: true,
			want:     []string{googledrive.DriveReadonlyScope, docs.DocumentsReadonlyScope},
		},
		{
			name:          "replaceScopes discards existing scopes",
			existing:      []string{googledrive.DriveReadonlyScope, sheets.SpreadsheetsScope},
			services:      []string{"slides"},
			readOnly:      true,
			replaceScopes: true,
			want:          []string{slides.PresentationsReadonlyScope},
		},
		{
			name:     "bare readonly resets to all readonly scopes",
			existing: DefaultOAuthScopes(),
			services: nil,
			readOnly: true,
			want:     ReadOnlyOAuthScopes(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ComputeLoginScopes(tc.existing, tc.services, tc.readOnly, tc.replaceScopes)
			if err != nil {
				t.Fatalf("ComputeLoginScopes(%v, %v, %t, %t) error = %v", tc.existing, tc.services, tc.readOnly, tc.replaceScopes, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ComputeLoginScopes(%v, %v, %t, %t) mismatch (-want +got):\n%s", tc.existing, tc.services, tc.readOnly, tc.replaceScopes, diff)
			}
		})
	}
}

func TestBuildAuthURL(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID:    "cid",
		RedirectURL: "http://127.0.0.1:43210",
		Scopes:      []string{googledrive.DriveReadonlyScope},
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://accounts.google.com/o/oauth2/auth",
		},
	}

	rawURL := buildAuthURL(cfg, "csrf-state-123", "pkce-verifier-value", true, true)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", rawURL, err)
	}
	q := parsed.Query()
	if got := q.Get("access_type"); got != "offline" {
		t.Errorf("buildAuthURL(cfg, true, true) access_type = %q, want %q", got, "offline")
	}
	if got := q.Get("code_challenge_method"); got != "S256" {
		t.Errorf("buildAuthURL(cfg, true, true) code_challenge_method = %q, want %q", got, "S256")
	}
	if got := q.Get("prompt"); got != "consent" {
		t.Errorf("buildAuthURL(cfg, true, true) prompt = %q, want %q", got, "consent")
	}
	if got := q.Get("include_granted_scopes"); got != "true" {
		t.Errorf("buildAuthURL(cfg, true, true) include_granted_scopes = %q, want %q", got, "true")
	}
}

func TestResolveAccessToken(t *testing.T) {
	t.Setenv(AccessTokenEnvVar, "")

	tok, src, err := ResolveAccessToken("", nil)
	if err != nil || tok != "" || src != "" {
		t.Fatalf("ResolveAccessToken(\"\", nil) = (%q, %q, %v), want empty", tok, src, err)
	}

	t.Setenv(AccessTokenEnvVar, "  env-access-token  ")
	tok, src, err = ResolveAccessToken("", nil)
	if err != nil || tok != "env-access-token" || src != "env:"+AccessTokenEnvVar {
		t.Fatalf("ResolveAccessToken(\"\", nil) with env = (%q, %q, %v), want (%q, %q, nil)", tok, src, err, "env-access-token", "env:"+AccessTokenEnvVar)
	}

	tokenPath := filepath.Join(t.TempDir(), "token.txt")
	if err := os.WriteFile(tokenPath, []byte("file-access-token\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", tokenPath, err)
	}
	tok, src, err = ResolveAccessToken(tokenPath, nil)
	if err != nil || tok != "file-access-token" || src != "file:"+tokenPath {
		t.Fatalf("ResolveAccessToken(%q, nil) = (%q, %q, %v), want (%q, %q, nil)", tokenPath, tok, src, err, "file-access-token", "file:"+tokenPath)
	}

	tok, src, err = ResolveAccessToken("-", strings.NewReader("stdin-access-token\n"))
	if err != nil || tok != "stdin-access-token" || src != "stdin" {
		t.Fatalf("ResolveAccessToken(\"-\", stdin) = (%q, %q, %v), want (%q, %q, nil)", tok, src, err, "stdin-access-token", "stdin")
	}

	if _, _, err := ResolveAccessToken("-", nil); err == nil {
		t.Errorf("ResolveAccessToken(\"-\", nil) error = nil, want non-nil")
	}
}

func TestRevokeToken(t *testing.T) {
	var gotMethod, gotURL, gotBody string
	httpClient := &http.Client{
		Transport: authTestRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotURL = req.URL.String()
			b, _ := io.ReadAll(req.Body)
			gotBody = string(b)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	tok := &oauth2.Token{AccessToken: "acc-1", RefreshToken: "ref-1"}
	if err := RevokeToken(t.Context(), httpClient, tok); err != nil {
		t.Fatalf("RevokeToken(ctx, client, %+v) error = %v", tok, err)
	}
	if gotMethod != http.MethodPost || gotURL != googleRevokeEndpoint || gotBody != "token=ref-1" {
		t.Errorf("RevokeToken request = (%s, %s, %q), want (POST, %s, %q)", gotMethod, gotURL, gotBody, googleRevokeEndpoint, "token=ref-1")
	}

	// Falls back to access token when refresh token is empty.
	accessOnly := &oauth2.Token{AccessToken: "acc-only"}
	if err := RevokeToken(t.Context(), httpClient, accessOnly); err != nil {
		t.Fatalf("RevokeToken(ctx, client, %+v) error = %v", accessOnly, err)
	}
	if gotBody != "token=acc-only" {
		t.Errorf("RevokeToken(accessOnly) body = %q, want %q", gotBody, "token=acc-only")
	}
}

func TestRefreshProfileTokenAndResetProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(encryptionKeyEnvVar, "")

	secretPath, err := ProfileSecretFile(AgentProfile)
	if err != nil {
		t.Fatalf("ProfileSecretFile(%q) error = %v", AgentProfile, err)
	}
	secretJSON := `{"installed":{"client_id":"cid","client_secret":"csec","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://localhost"]}}`
	if err := os.WriteFile(secretPath, []byte(secretJSON), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", secretPath, err)
	}

	tokPath, err := ProfileTokenCacheFile(AgentProfile)
	if err != nil {
		t.Fatalf("ProfileTokenCacheFile(%q) error = %v", AgentProfile, err)
	}
	if err := SaveToken(tokPath, &oauth2.Token{
		AccessToken:  "old-access",
		RefreshToken: "refresh-abc",
		Expiry:       time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveToken(%q) error = %v", tokPath, err)
	}

	var revoked bool
	fakeClient := &http.Client{
		Transport: authTestRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == googleRevokeEndpoint {
				revoked = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"new-refreshed-access","token_type":"Bearer","expires_in":3600}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Request:    req,
			}, nil
		}),
	}
	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, fakeClient)

	refreshed, err := RefreshProfileToken(ctx, AgentProfile, "")
	if err != nil {
		t.Fatalf("RefreshProfileToken(ctx, %q, \"\") error = %v", AgentProfile, err)
	}
	if refreshed.AccessToken != "new-refreshed-access" || refreshed.RefreshToken != "refresh-abc" {
		t.Fatalf("RefreshProfileToken(ctx, %q, \"\") = %+v, want access=%q refresh=%q", AgentProfile, refreshed, "new-refreshed-access", "refresh-abc")
	}

	res, err := ResetProfile(ctx, AgentProfile, false)
	if err != nil {
		t.Fatalf("ResetProfile(ctx, %q, false) error = %v", AgentProfile, err)
	}
	if !revoked || !res.Revoked {
		t.Errorf("ResetProfile(ctx, %q, false) revoked = (%t, %t), want (true, true)", AgentProfile, revoked, res.Revoked)
	}
	if len(res.RemovedFiles) < 2 {
		t.Errorf("ResetProfile(ctx, %q, false) RemovedFiles = %v, want at least 2 files removed", AgentProfile, res.RemovedFiles)
	}
}
