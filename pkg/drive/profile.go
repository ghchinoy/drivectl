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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// DefaultProfile is the standard interactive CLI OAuth profile.
	DefaultProfile = "default"
	// AgentProfile is the dedicated OAuth profile for AI agent / MCP runs.
	AgentProfile = "agent"

	profileMetadataFile = "profile.json"
)

var validProfilePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ProfileMetadata stores non-secret metadata for an authentication profile.
type ProfileMetadata struct {
	Profile   string   `json:"profile"`
	Agentic   bool     `json:"agentic"`
	ClientID  string   `json:"client_id,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// AuthStatus summarizes the current authentication state for a profile.
type AuthStatus struct {
	Profile           string             `json:"profile"`
	ConfigDir         string             `json:"config_dir"`
	SecretFile        string             `json:"secret_file"`
	SecretExists      bool               `json:"secret_exists"`
	ClientID          string             `json:"client_id,omitempty"`
	Agentic           bool               `json:"agentic"`
	BearerTokenSource string             `json:"bearer_token_source,omitempty"`
	Token             TokenStoreMetadata `json:"token"`
	TokenValid        bool               `json:"token_valid"`
	TokenExpiry       string             `json:"token_expiry,omitempty"`
	HasRefresh        bool               `json:"has_refresh_token"`
	Scopes            []string           `json:"scopes,omitempty"`
}

// AuthOptions configures profile-aware OAuth authentication and request tagging.
type AuthOptions struct {
	SecretFile           string
	Profile              string
	Agentic              bool
	NoBrowserAuth        bool
	UserAgent            string
	Scopes               []string
	PromptConsent        bool
	IncludeGrantedScopes bool
}

// ResetResult summarizes the outcome of resetting an authentication profile.
type ResetResult struct {
	Profile       string   `json:"profile"`
	ConfigDir     string   `json:"config_dir"`
	Revoked       bool     `json:"revoked"`
	RevokeWarning string   `json:"revoke_warning,omitempty"`
	RemovedFiles  []string `json:"removed_files,omitempty"`
}

// NormalizeProfile validates and normalizes a profile name, defaulting empty names to "default".
func NormalizeProfile(profile string) (string, error) {
	trimmed := strings.TrimSpace(profile)
	if trimmed == "" || trimmed == DefaultProfile {
		return DefaultProfile, nil
	}
	if !validProfilePattern.MatchString(trimmed) {
		return "", fmt.Errorf("invalid profile name %q: use letters, digits, hyphens, or underscores", profile)
	}
	return trimmed, nil
}

// ProfileConfigDir returns the configuration directory for the specified profile.
// The default profile maps directly to ~/.config/drivectl for backwards compatibility.
func ProfileConfigDir(profile string) (string, error) {
	norm, err := NormalizeProfile(profile)
	if err != nil {
		return "", err
	}
	baseDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	if norm == DefaultProfile {
		return baseDir, nil
	}
	profileDir := filepath.Join(baseDir, "profiles", norm)
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create profile directory %q: %w", profileDir, err)
	}
	return profileDir, nil
}

// ProfileTokenCacheFile returns the token cache path for the specified profile.
func ProfileTokenCacheFile(profile string) (string, error) {
	dir, err := ProfileConfigDir(profile)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

// ProfileSecretFile returns the cached client_secret.json path for the specified profile.
func ProfileSecretFile(profile string) (string, error) {
	dir, err := ProfileConfigDir(profile)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "client_secret.json"), nil
}

// SaveProfileMetadata writes non-secret profile metadata (including agentic client labeling).
func SaveProfileMetadata(profile string, meta ProfileMetadata) error {
	dir, err := ProfileConfigDir(profile)
	if err != nil {
		return err
	}
	norm, err := NormalizeProfile(profile)
	if err != nil {
		return err
	}
	meta.Profile = norm
	meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding profile metadata: %w", err)
	}
	return writeFileAtomic(filepath.Join(dir, profileMetadataFile), append(data, '\n'), 0o600)
}

// LoadProfileMetadata loads profile metadata if present, returning a zero-value default otherwise.
func LoadProfileMetadata(profile string) (ProfileMetadata, error) {
	norm, err := NormalizeProfile(profile)
	if err != nil {
		return ProfileMetadata{}, err
	}
	dir, err := ProfileConfigDir(norm)
	if err != nil {
		return ProfileMetadata{}, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, profileMetadataFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProfileMetadata{Profile: norm}, nil
		}
		return ProfileMetadata{}, fmt.Errorf("reading profile metadata: %w", err)
	}
	var meta ProfileMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ProfileMetadata{}, fmt.Errorf("parsing profile metadata: %w", err)
	}
	if meta.Profile == "" {
		meta.Profile = norm
	}
	return meta, nil
}

// ExtractClientID parses the non-secret OAuth client_id from a Google client_secret.json payload.
func ExtractClientID(secretJSON []byte) string {
	var parsed struct {
		Installed struct {
			ClientID string `json:"client_id"`
		} `json:"installed"`
		Web struct {
			ClientID string `json:"client_id"`
		} `json:"web"`
	}
	if err := json.Unmarshal(secretJSON, &parsed); err != nil {
		return ""
	}
	if parsed.Installed.ClientID != "" {
		return parsed.Installed.ClientID
	}
	return parsed.Web.ClientID
}

// InspectAuthStatus inspects the local configuration and cached token for a profile without triggering login.
func InspectAuthStatus(profile string) (AuthStatus, error) {
	norm, err := NormalizeProfile(profile)
	if err != nil {
		return AuthStatus{}, err
	}
	dir, err := ProfileConfigDir(norm)
	if err != nil {
		return AuthStatus{}, err
	}

	secretPath := filepath.Join(dir, "client_secret.json")
	tokenPath := filepath.Join(dir, "token.json")
	meta, err := LoadProfileMetadata(norm)
	if err != nil {
		return AuthStatus{}, err
	}

	status := AuthStatus{
		Profile:    norm,
		ConfigDir:  dir,
		SecretFile: secretPath,
		ClientID:   meta.ClientID,
		Agentic:    meta.Agentic,
		Token:      InspectTokenFile(tokenPath),
		Scopes:     meta.Scopes,
	}

	if strings.TrimSpace(os.Getenv(AccessTokenEnvVar)) != "" {
		status.BearerTokenSource = "env:" + AccessTokenEnvVar
	}

	if rawSecret, err := os.ReadFile(secretPath); err == nil {
		status.SecretExists = true
		if status.ClientID == "" {
			status.ClientID = ExtractClientID(rawSecret)
		}
	}

	if tok, err := TokenFromFile(tokenPath); err == nil {
		status.TokenValid = tok.Valid()
		status.HasRefresh = tok.RefreshToken != ""
		if !tok.Expiry.IsZero() {
			status.TokenExpiry = tok.Expiry.UTC().Format(time.RFC3339)
		}
	}

	return status, nil
}

// ClearProfileToken removes the cached token file for the specified profile.
func ClearProfileToken(profile string) (string, error) {
	tokenPath, err := ProfileTokenCacheFile(profile)
	if err != nil {
		return "", err
	}
	if err := os.Remove(tokenPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return tokenPath, fmt.Errorf("removing cached token %q: %w", tokenPath, err)
	}
	return tokenPath, nil
}

// ResetProfile optionally revokes the cached OAuth token and removes all local credential
// and profile files for the specified profile.
func ResetProfile(ctx context.Context, profile string, skipRevoke bool) (ResetResult, error) {
	norm, err := NormalizeProfile(profile)
	if err != nil {
		return ResetResult{}, err
	}
	dir, err := ProfileConfigDir(norm)
	if err != nil {
		return ResetResult{}, err
	}

	res := ResetResult{
		Profile:   norm,
		ConfigDir: dir,
	}

	tokenPath := filepath.Join(dir, "token.json")
	if !skipRevoke {
		revokeCachedToken(ctx, tokenPath, &res)
	}

	targets := []string{
		tokenPath,
		filepath.Join(dir, "client_secret.json"),
		filepath.Join(dir, profileMetadataFile),
		filepath.Join(dir, encryptionKeyFile),
	}
	for _, targetPath := range targets {
		err := os.Remove(targetPath)
		if err == nil {
			res.RemovedFiles = append(res.RemovedFiles, targetPath)
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return res, fmt.Errorf("removing %q: %w", targetPath, err)
		}
	}

	if norm != DefaultProfile {
		_ = os.Remove(dir)
	}
	return res, nil
}

func revokeCachedToken(ctx context.Context, tokenPath string, res *ResetResult) {
	tok, err := TokenFromFile(tokenPath)
	if err != nil {
		return
	}
	if revokeErr := RevokeToken(ctx, nil, tok); revokeErr != nil {
		res.RevokeWarning = revokeErr.Error()
		return
	}
	res.Revoked = true
}

// WrapHTTPClientUserAgent wraps an *http.Client's transport to attach a default User-Agent header.
func WrapHTTPClientUserAgent(client *http.Client, userAgent string) *http.Client {
	if client == nil || strings.TrimSpace(userAgent) == "" {
		return client
	}
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	cloned := *client
	cloned.Transport = &userAgentTransport{
		base:      base,
		userAgent: userAgent,
	}
	return &cloned
}

type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if clone.Header.Get("User-Agent") == "" {
		clone.Header.Set("User-Agent", t.userAgent)
	}
	return t.base.RoundTrip(clone)
}
