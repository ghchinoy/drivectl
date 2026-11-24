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

package mcp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/viper"

	"github.com/ghchinoy/drivectl/pkg/drive"
)

func TestNormalizeMCPHTTPAddr(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{":8080", "127.0.0.1:8080"},
		{" :9090 ", "127.0.0.1:9090"},
		{"127.0.0.1:8080", "127.0.0.1:8080"},
		{"0.0.0.0:8080", "0.0.0.0:8080"},
		{"localhost:8080", "localhost:8080"},
	}

	for _, tc := range tests {
		if got := normalizeMCPHTTPAddr(tc.in); got != tc.want {
			t.Errorf("normalizeMCPHTTPAddr(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestWrapMCPHTTPSecurity(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		authToken  string
		origin     string
		authHeader string
		wantStatus int
	}{
		{
			name:       "no origin and no auth configured",
			wantStatus: http.StatusOK,
		},
		{
			name:       "localhost origin allowed",
			origin:     "http://localhost:3000",
			wantStatus: http.StatusOK,
		},
		{
			name:       "ipv4 loopback origin allowed",
			origin:     "http://127.0.0.1:8080",
			wantStatus: http.StatusOK,
		},
		{
			name:       "ipv6 loopback origin allowed",
			origin:     "http://[::1]:8080",
			wantStatus: http.StatusOK,
		},
		{
			name:       "external origin rejected",
			origin:     "https://evil.example.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid scheme origin rejected",
			origin:     "javascript://localhost",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "missing bearer token rejected when configured",
			authToken:  "secret-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong bearer token rejected",
			authToken:  "secret-token",
			authHeader: "Bearer wrong-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid bearer token accepted",
			authToken:  "secret-token",
			origin:     "http://localhost:8080",
			authHeader: "Bearer secret-token",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := wrapMCPHTTPSecurity(okHandler, tc.authToken)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatalf("wrapMCPHTTPSecurity(%q) WWW-Authenticate header = empty, want non-empty on 401", tc.name)
			}
		})
	}
}

func TestResolveMCPProfileAndSecret(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DRIVECTL_AGENT_SECRETS", "")
	viper.Reset()
	defer viper.Reset()

	// Default fallback when no agent profile exists.
	prof, _ := resolveMCPProfileAndSecret()
	if prof != drive.DefaultProfile {
		t.Fatalf("profile = %q, want %q", prof, drive.DefaultProfile)
	}

	// Auto-select agent profile when profiles/agent/client_secret.json exists.
	agentSecretPath, err := drive.ProfileSecretFile(drive.AgentProfile)
	if err != nil {
		t.Fatalf("ProfileSecretFile(agent) error = %v", err)
	}
	if err := os.WriteFile(agentSecretPath, []byte(`{"installed":{"client_id":"abc"}}`), 0o600); err != nil {
		t.Fatalf("writing agent secret error = %v", err)
	}
	prof, sec := resolveMCPProfileAndSecret()
	if prof != drive.AgentProfile || sec != agentSecretPath {
		t.Fatalf("resolveMCPProfileAndSecret() = (%q, %q), want (%q, %q)", prof, sec, drive.AgentProfile, agentSecretPath)
	}

	// Explicit DRIVECTL_AGENT_SECRETS takes precedence over cached agent secret path.
	t.Setenv("DRIVECTL_AGENT_SECRETS", "/tmp/custom-agent-secret.json")
	prof, sec = resolveMCPProfileAndSecret()
	if prof != drive.AgentProfile || sec != "/tmp/custom-agent-secret.json" {
		t.Fatalf("resolveMCPProfileAndSecret() with env = (%q, %q), want (%q, %q)", prof, sec, drive.AgentProfile, "/tmp/custom-agent-secret.json")
	}
}
