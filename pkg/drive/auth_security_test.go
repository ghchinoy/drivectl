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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSaveToken_EncryptedAtRestAndLegacyMigration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(encryptionKeyEnvVar, "")

	cachePath, err := TokenCacheFile()
	if err != nil {
		t.Fatalf("TokenCacheFile() error = %v", err)
	}

	wantTok := &oauth2.Token{
		AccessToken:  "super-secret-access-token",
		TokenType:    "Bearer",
		RefreshToken: "super-secret-refresh-token",
		Expiry:       time.Now().Add(time.Hour).UTC().Truncate(time.Second),
	}

	if err := SaveToken(cachePath, wantTok); err != nil {
		t.Fatalf("SaveToken(%q) error = %v", cachePath, err)
	}

	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", cachePath, err)
	}
	if strings.Contains(string(raw), wantTok.AccessToken) || strings.Contains(string(raw), wantTok.RefreshToken) {
		t.Fatalf("SaveToken(%q) wrote plaintext secret bytes: %s", cachePath, string(raw))
	}

	meta := InspectTokenFile(cachePath)
	if !meta.Exists || !meta.Encrypted {
		t.Fatalf("InspectTokenFile(%q) = %+v, want Exists=true Encrypted=true", cachePath, meta)
	}

	gotTok, err := TokenFromFile(cachePath)
	if err != nil {
		t.Fatalf("TokenFromFile(%q) error = %v", cachePath, err)
	}
	if gotTok.AccessToken != wantTok.AccessToken || gotTok.RefreshToken != wantTok.RefreshToken {
		t.Fatalf("TokenFromFile(%q) = %+v, want %+v", cachePath, gotTok, wantTok)
	}

	// Verify legacy plaintext token files still read cleanly and upgrade in place.
	legacyPath := filepath.Join(t.TempDir(), "token.json")
	legacyBytes, _ := json.Marshal(wantTok)
	if err := os.WriteFile(legacyPath, legacyBytes, 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", legacyPath, err)
	}
	if InspectTokenFile(legacyPath).Encrypted {
		t.Fatalf("InspectTokenFile(%q).Encrypted = true, want false for legacy plaintext", legacyPath)
	}
	legacyTok, err := TokenFromFile(legacyPath)
	if err != nil || legacyTok.AccessToken != wantTok.AccessToken {
		t.Fatalf("TokenFromFile(%q) = %+v, %v; want access token %q", legacyPath, legacyTok, err, wantTok.AccessToken)
	}

	if _, err := fetchClientWithCache(t.Context(), &oauth2.Config{}, legacyPath, true); err != nil {
		t.Fatalf("fetchClientWithCache(%q) error = %v", legacyPath, err)
	}
	if !InspectTokenFile(legacyPath).Encrypted {
		t.Fatalf("InspectTokenFile(%q).Encrypted = false after fetchClientWithCache, want true", legacyPath)
	}
}

func TestEncryptionKeyFromEnv(t *testing.T) {
	key := make([]byte, aes256KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("io.ReadFull(rand.Reader) error = %v", err)
	}
	t.Setenv(encryptionKeyEnvVar, base64.StdEncoding.EncodeToString(key))

	tokPath := filepath.Join(t.TempDir(), "token.json")
	tok := &oauth2.Token{AccessToken: "env-key-access", TokenType: "Bearer"}
	if err := SaveToken(tokPath, tok); err != nil {
		t.Fatalf("SaveToken(%q) error = %v", tokPath, err)
	}
	meta := InspectTokenFile(tokPath)
	if !meta.Encrypted || meta.KeySource != "env:"+encryptionKeyEnvVar {
		t.Fatalf("InspectTokenFile(%q) = %+v, want Encrypted=true KeySource=env:%s", tokPath, meta, encryptionKeyEnvVar)
	}
	got, err := TokenFromFile(tokPath)
	if err != nil || got.AccessToken != tok.AccessToken {
		t.Fatalf("TokenFromFile(%q) = %+v, %v", tokPath, got, err)
	}

	// Wrong key length should fail.
	t.Setenv(encryptionKeyEnvVar, base64.StdEncoding.EncodeToString([]byte("too-short")))
	if _, err := TokenFromFile(tokPath); err == nil {
		t.Fatalf("TokenFromFile(%q) with invalid key length succeeded, want error", tokPath)
	}
}

func TestPersistingTokenSource_PersistsRotatedTokens(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(encryptionKeyEnvVar, "")

	tokPath := filepath.Join(t.TempDir(), "token.json")
	initial := &oauth2.Token{AccessToken: "old-access", RefreshToken: "refresh-1"}
	rotated := &oauth2.Token{AccessToken: "new-access", RefreshToken: "refresh-2"}

	ts := newPersistingTokenSource(oauth2.StaticTokenSource(rotated), tokPath, initial)
	got, err := ts.Token()
	if err != nil {
		t.Fatalf("ts.Token() error = %v", err)
	}
	if got.AccessToken != "new-access" {
		t.Fatalf("ts.Token().AccessToken = %q, want %q", got.AccessToken, "new-access")
	}

	persisted, err := TokenFromFile(tokPath)
	if err != nil {
		t.Fatalf("TokenFromFile(%q) error = %v", tokPath, err)
	}
	if persisted.AccessToken != "new-access" || persisted.RefreshToken != "refresh-2" {
		t.Fatalf("TokenFromFile(%q) = %+v, want %+v", tokPath, persisted, rotated)
	}
}

func TestOAuthCallbackHandler_CSRFStateAndErrors(t *testing.T) {
	state, err := generateOAuthState()
	if err != nil || len(state) < 32 {
		t.Fatalf("generateOAuthState() = %q, %v", state, err)
	}
	state2, err := generateOAuthState()
	if err != nil || state == state2 {
		t.Fatalf("generateOAuthState() should return unique values")
	}

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCode   string
		wantErrSub string
	}{
		{
			name:       "valid state and code",
			query:      "/?state=" + state + "&code=auth-code-123",
			wantStatus: http.StatusOK,
			wantCode:   "auth-code-123",
		},
		{
			name:       "mismatched state rejected",
			query:      "/?state=wrong-state&code=auth-code-123",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "state mismatch",
		},
		{
			name:       "missing state rejected",
			query:      "/?code=auth-code-123",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "state mismatch",
		},
		{
			name:       "missing code rejected",
			query:      "/?state=" + state,
			wantStatus: http.StatusBadRequest,
			wantErrSub: "no code",
		},
		{
			name:       "oauth error rejected",
			query:      "/?error=access_denied",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "access_denied",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			codeChan := make(chan string, 1)
			errChan := make(chan error, 1)
			handler := newOAuthCallbackHandler(state, codeChan, errChan)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.query, nil)
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantCode != "" {
				select {
				case got := <-codeChan:
					if got != tc.wantCode {
						t.Fatalf("code = %q, want %q", got, tc.wantCode)
					}
				default:
					t.Fatalf("expected code on codeChan")
				}
			}
			if tc.wantErrSub != "" {
				select {
				case gotErr := <-errChan:
					if !strings.Contains(gotErr.Error(), tc.wantErrSub) {
						t.Fatalf("err = %q, want substring %q", gotErr.Error(), tc.wantErrSub)
					}
				default:
					t.Fatalf("expected error on errChan")
				}
			}
		})
	}
}

func TestProfilesAndAuthStatus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(encryptionKeyEnvVar, "")

	if _, err := NormalizeProfile("../escape"); err == nil {
		t.Fatalf("NormalizeProfile(../escape) succeeded, want error")
	}

	agentDir, err := ProfileConfigDir(AgentProfile)
	if err != nil {
		t.Fatalf("ProfileConfigDir(agent) error = %v", err)
	}
	if !strings.HasSuffix(agentDir, filepath.Join("profiles", "agent")) {
		t.Fatalf("agentDir = %q, want suffix profiles/agent", agentDir)
	}

	secretJSON := []byte(`{"installed":{"client_id":"12345.apps.googleusercontent.com","client_secret":"sec"}}`)
	secretPath, _ := ProfileSecretFile(AgentProfile)
	if err := os.WriteFile(secretPath, secretJSON, 0o600); err != nil {
		t.Fatalf("writing secret error = %v", err)
	}

	if err := SaveProfileMetadata(AgentProfile, ProfileMetadata{
		Agentic:  true,
		ClientID: ExtractClientID(secretJSON),
		Scopes:   DefaultOAuthScopes(),
	}); err != nil {
		t.Fatalf("SaveProfileMetadata(%q) error = %v", AgentProfile, err)
	}

	tokPath, _ := ProfileTokenCacheFile(AgentProfile)
	if err := SaveToken(tokPath, &oauth2.Token{
		AccessToken:  "agent-tok",
		RefreshToken: "agent-refresh",
		Expiry:       time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveToken(%q) error = %v", tokPath, err)
	}

	status, err := InspectAuthStatus(AgentProfile)
	if err != nil {
		t.Fatalf("InspectAuthStatus(%q) error = %v", AgentProfile, err)
	}
	if status.Profile != AgentProfile || !status.Agentic || !status.SecretExists || !status.Token.Encrypted || !status.TokenValid || !status.HasRefresh {
		t.Fatalf("InspectAuthStatus(%q) = %+v, unexpected fields", AgentProfile, status)
	}
	if status.ClientID != "12345.apps.googleusercontent.com" {
		t.Fatalf("InspectAuthStatus(%q).ClientID = %q, want 12345.apps.googleusercontent.com", AgentProfile, status.ClientID)
	}

	if _, err := ClearProfileToken(AgentProfile); err != nil {
		t.Fatalf("ClearProfileToken(%q) error = %v", AgentProfile, err)
	}
	statusAfter, err := InspectAuthStatus(AgentProfile)
	if err != nil || statusAfter.Token.Exists {
		t.Fatalf("InspectAuthStatus(%q) after logout = %+v, err = %v; want Token.Exists=false", AgentProfile, statusAfter, err)
	}
}

type authTestRoundTripFunc func(*http.Request) (*http.Response, error)

func (f authTestRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWrapHTTPClientUserAgent(t *testing.T) {
	var gotUA string
	base := &http.Client{
		Transport: authTestRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotUA = req.Header.Get("User-Agent")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	wrapped := WrapHTTPClientUserAgent(base, "drivectl/test (mcp; agentic=true)")
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", nil)
	resp, err := wrapped.Do(req)
	if err != nil {
		t.Fatalf("wrapped.Do() error = %v", err)
	}
	_ = resp.Body.Close()
	if gotUA != "drivectl/test (mcp; agentic=true)" {
		t.Fatalf("User-Agent = %q, want %q", gotUA, "drivectl/test (mcp; agentic=true)")
	}
}
