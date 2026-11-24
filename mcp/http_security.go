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
	"crypto/subtle"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const mcpAuthTokenEnvVar = "DRIVECTL_MCP_AUTH_TOKEN"

// normalizeMCPHTTPAddr defaults hostless addresses (such as ":8080") to loopback ("127.0.0.1:8080")
// so local MCP HTTP servers are not accidentally exposed on all network interfaces.
func normalizeMCPHTTPAddr(addr string) string {
	trimmed := strings.TrimSpace(addr)
	if strings.HasPrefix(trimmed, ":") {
		return "127.0.0.1" + trimmed
	}
	return trimmed
}

func isLoopbackHost(host string) bool {
	h := strings.TrimSpace(strings.ToLower(host))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(h, "[]"))
	return ip != nil && ip.IsLoopback()
}

func isBindAddrLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	return isLoopbackHost(host)
}

func validateMCPOrigin(origin string) bool {
	if strings.TrimSpace(origin) == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return isLoopbackHost(parsed.Hostname())
}

func validateBearerAuth(authHeader, expectedToken string) bool {
	if expectedToken == "" {
		return true
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}
	provided := strings.TrimPrefix(authHeader, prefix)
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) == 1
}

// wrapMCPHTTPSecurity enforces Origin validation (preventing DNS rebinding from browser pages)
// and optional Bearer token authentication when DRIVECTL_MCP_AUTH_TOKEN is configured.
func wrapMCPHTTPSecurity(next http.Handler, authToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validateMCPOrigin(r.Header.Get("Origin")) {
			http.Error(w, "Forbidden: invalid Origin header", http.StatusForbidden)
			return
		}
		if !validateBearerAuth(r.Header.Get("Authorization"), authToken) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="drivectl-mcp"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logNonLoopbackWarning(addr, authToken string) {
	if isBindAddrLoopback(addr) {
		return
	}
	if authToken == "" {
		log.Printf("WARNING: MCP HTTP server bound to non-loopback address %s without %s configured", addr, mcpAuthTokenEnvVar)
		return
	}
	log.Printf("MCP HTTP server bound to non-loopback address %s with Bearer authentication enabled", addr)
}
