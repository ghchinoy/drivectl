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

// Package drive provides Google Drive, Docs, Sheets, and Slides API client wrappers.
package drive

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"
)

// DefaultOAuthScopes returns the standard set of Google Workspace OAuth scopes requested by drivectl.
func DefaultOAuthScopes() []string {
	return []string{
		drive.DriveReadonlyScope,
		docs.DocumentsReadonlyScope,
		sheets.SpreadsheetsScope,
		slides.PresentationsScope,
		drive.DriveFileScope,
	}
}

// Services bundles pre-configured Google Workspace API clients sharing an HTTP client.
type Services struct {
	Client *http.Client
	Drive  *drive.Service
	Docs   *docs.Service
	Sheets *sheets.Service
	Slides *slides.Service
}

// NewClientFromTokenSource returns an *http.Client backed by the provided oauth2.TokenSource.
// It performs no disk I/O and writes nothing to stdout.
func NewClientFromTokenSource(ctx context.Context, ts oauth2.TokenSource) *http.Client {
	return oauth2.NewClient(ctx, ts)
}

// NewClientFromBearerToken returns a stateless *http.Client that attaches the given OAuth2 access token.
// It performs no disk I/O and writes nothing to stdout.
func NewClientFromBearerToken(ctx context.Context, accessToken string) *http.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	})
	return NewClientFromTokenSource(ctx, ts)
}

// NewDriveService creates a *drive.Service backed by client.
func NewDriveService(ctx context.Context, client *http.Client) (*drive.Service, error) {
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create drive service: %w", err)
	}
	return srv, nil
}

// NewDocsService creates a *docs.Service backed by client.
func NewDocsService(ctx context.Context, client *http.Client) (*docs.Service, error) {
	srv, err := docs.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create docs service: %w", err)
	}
	return srv, nil
}

// NewSheetsService creates a *sheets.Service backed by client.
func NewSheetsService(ctx context.Context, client *http.Client) (*sheets.Service, error) {
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create sheets service: %w", err)
	}
	return srv, nil
}

// NewSlidesService creates a *slides.Service backed by client.
func NewSlidesService(ctx context.Context, client *http.Client) (*slides.Service, error) {
	srv, err := slides.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create slides service: %w", err)
	}
	return srv, nil
}

// NewServicesFromClient initializes Drive, Docs, Sheets, and Slides services from a single *http.Client.
func NewServicesFromClient(ctx context.Context, client *http.Client) (*Services, error) {
	driveSvc, err := NewDriveService(ctx, client)
	if err != nil {
		return nil, err
	}
	docsSvc, err := NewDocsService(ctx, client)
	if err != nil {
		return nil, err
	}
	sheetsSvc, err := NewSheetsService(ctx, client)
	if err != nil {
		return nil, err
	}
	slidesSvc, err := NewSlidesService(ctx, client)
	if err != nil {
		return nil, err
	}
	return &Services{
		Client: client,
		Drive:  driveSvc,
		Docs:   docsSvc,
		Sheets: sheetsSvc,
		Slides: slidesSvc,
	}, nil
}

// ConfigDir returns the path to the application's config directory.
func ConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "drivectl")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return configDir, nil
}

var googleRevokeEndpoint = "https://oauth2.googleapis.com/revoke"

// ResolveAccessToken resolves a pre-obtained OAuth 2.0 bearer token from tokenFile (or stdin when
// tokenFile is "-") or from the DRIVECTL_TOKEN environment variable. Returns ("", "", nil) when
// neither source is configured.
func ResolveAccessToken(tokenFile string, stdin io.Reader) (token, source string, err error) {
	trimmedFile := strings.TrimSpace(tokenFile)
	if trimmedFile != "" {
		return readTokenFromFileOrStdin(trimmedFile, stdin)
	}
	if envToken := strings.TrimSpace(os.Getenv(AccessTokenEnvVar)); envToken != "" {
		return envToken, "env:" + AccessTokenEnvVar, nil
	}
	return "", "", nil
}

func readTokenFromFileOrStdin(tokenFile string, stdin io.Reader) (string, string, error) {
	if tokenFile == "-" {
		if stdin == nil {
			return "", "", errors.New("reading access token from stdin is unavailable in this mode")
		}
		raw, err := io.ReadAll(stdin)
		if err != nil {
			return "", "", fmt.Errorf("reading access token from stdin: %w", err)
		}
		tok := strings.TrimSpace(string(raw))
		if tok == "" {
			return "", "", errors.New("access token from stdin is empty")
		}
		return tok, "stdin", nil
	}

	raw, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", "", fmt.Errorf("reading access token file %q: %w", tokenFile, err)
	}
	tok := strings.TrimSpace(string(raw))
	if tok == "" {
		return "", "", fmt.Errorf("access token file %q is empty", tokenFile)
	}
	return tok, "file:" + tokenFile, nil
}

// RevokeToken revokes an OAuth 2.0 token (preferring its refresh token) at Google's revocation endpoint.
func RevokeToken(ctx context.Context, httpClient *http.Client, token *oauth2.Token) error {
	revokeValue := tokenValueForRevoke(token)
	if revokeValue == "" {
		return nil
	}

	form := url.Values{"token": {revokeValue}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleRevokeEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("creating revoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := resolveRevokeHTTPClient(ctx, httpClient).Do(req)
	if err != nil {
		return fmt.Errorf("revoking OAuth token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("OAuth revocation returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func tokenValueForRevoke(token *oauth2.Token) string {
	if token == nil {
		return ""
	}
	if refresh := strings.TrimSpace(token.RefreshToken); refresh != "" {
		return refresh
	}
	return strings.TrimSpace(token.AccessToken)
}

func resolveRevokeHTTPClient(ctx context.Context, httpClient *http.Client) *http.Client {
	if httpClient != nil {
		return httpClient
	}
	if ctxClient, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && ctxClient != nil {
		return ctxClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func loadProfileOAuthConfig(profile, secretFile string, explicitScopes []string) (*oauth2.Config, string, error) {
	norm, err := NormalizeProfile(profile)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(secretFile) == "" {
		secretFile, err = ProfileSecretFile(norm)
		if err != nil {
			return nil, "", err
		}
	}

	rawSecret, err := os.ReadFile(secretFile)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read client secret file at %q: %w", secretFile, err)
	}

	scopes := resolveProfileScopes(norm, explicitScopes)
	config, err := google.ConfigFromJSON(rawSecret, scopes...)
	if err != nil {
		return nil, "", fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	cacheFile, err := ProfileTokenCacheFile(norm)
	if err != nil {
		return nil, "", fmt.Errorf("unable to get path to cached credential file: %w", err)
	}
	return config, cacheFile, nil
}

func resolveProfileScopes(profile string, explicitScopes []string) []string {
	if len(explicitScopes) > 0 {
		return explicitScopes
	}
	if meta, err := LoadProfileMetadata(profile); err == nil && len(meta.Scopes) > 0 {
		return meta.Scopes
	}
	return DefaultOAuthScopes()
}

// RefreshProfileToken forces an immediate refresh of the cached OAuth token for profile
// and persists the refreshed encrypted token to disk.
func RefreshProfileToken(ctx context.Context, profile, secretFile string) (*oauth2.Token, error) {
	config, cacheFile, err := loadProfileOAuthConfig(profile, secretFile, nil)
	if err != nil {
		return nil, err
	}

	tok, err := TokenFromFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("reading cached token %q: %w", cacheFile, err)
	}
	if strings.TrimSpace(tok.RefreshToken) == "" {
		return nil, errors.New("cached token has no refresh token; re-run 'drivectl auth login --prompt-consent'")
	}

	expiredCopy := *tok
	expiredCopy.Expiry = time.Now().Add(-time.Minute)
	refreshed, err := config.TokenSource(ctx, &expiredCopy).Token()
	if err != nil {
		return nil, fmt.Errorf("refreshing OAuth token: %w", err)
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = tok.RefreshToken
	}

	if err := SaveToken(cacheFile, refreshed); err != nil {
		return nil, err
	}
	return refreshed, nil
}

// NewOAuthClient creates a new HTTP client with OAuth 2.0 authentication using the default profile.
func NewOAuthClient(ctx context.Context, secretFile string, noBrowserAuth bool) (*http.Client, error) {
	return NewOAuthClientWithOptions(ctx, AuthOptions{
		SecretFile:    secretFile,
		Profile:       DefaultProfile,
		NoBrowserAuth: noBrowserAuth,
	})
}

// NewOAuthClientWithOptions creates a profile-aware HTTP client with OAuth 2.0 authentication.
func NewOAuthClientWithOptions(ctx context.Context, opts AuthOptions) (*http.Client, error) {
	config, cacheFile, err := loadProfileOAuthConfig(opts.Profile, opts.SecretFile, opts.Scopes)
	if err != nil {
		return nil, err
	}

	client, err := fetchClientWithCacheOptions(ctx, config, cacheFile, opts)
	if err != nil {
		return nil, err
	}
	return WrapHTTPClientUserAgent(client, opts.UserAgent), nil
}

// FetchClient retrieves a token from the default local cache or the web, then returns an authenticated HTTP client.
func FetchClient(ctx context.Context, config *oauth2.Config, noBrowserAuth bool) (*http.Client, error) {
	cacheFile, err := TokenCacheFile()
	if err != nil {
		return nil, fmt.Errorf("unable to get path to cached credential file: %w", err)
	}
	return fetchClientWithCache(ctx, config, cacheFile, noBrowserAuth)
}

func fetchClientWithCache(ctx context.Context, config *oauth2.Config, cacheFile string, noBrowserAuth bool) (*http.Client, error) {
	return fetchClientWithCacheOptions(ctx, config, cacheFile, AuthOptions{NoBrowserAuth: noBrowserAuth})
}

func fetchClientWithCacheOptions(ctx context.Context, config *oauth2.Config, cacheFile string, opts AuthOptions) (*http.Client, error) {
	tok, err := TokenFromFile(cacheFile)
	if err != nil {
		tok, err = fetchOAuthTokenOptions(ctx, config, opts)
		if err != nil {
			return nil, err
		}
		if err := SaveToken(cacheFile, tok); err != nil {
			return nil, err
		}
	}
	if err == nil && !InspectTokenFile(cacheFile).Encrypted {
		// Upgrade legacy plaintext token cache in place to AES-256-GCM.
		_ = SaveToken(cacheFile, tok)
	}

	ts := newPersistingTokenSource(config.TokenSource(ctx, tok), cacheFile, tok)
	return oauth2.NewClient(ctx, ts), nil
}

func buildAuthURL(config *oauth2.Config, state, verifier string, promptConsent, includeGrantedScopes bool) string {
	urlOpts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	}
	if promptConsent {
		urlOpts = append(urlOpts, oauth2.ApprovalForce)
	}
	if includeGrantedScopes {
		urlOpts = append(urlOpts, oauth2.SetAuthURLParam("include_granted_scopes", "true"))
	}
	return config.AuthCodeURL(state, urlOpts...)
}

// FetchTokenFromWebWithBrowser uses an ephemeral loopback web server with PKCE (S256)
// and CSRF state verification to handle the OAuth 2.0 callback flow.
func FetchTokenFromWebWithBrowser(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	return fetchTokenFromWebWithBrowserOptions(ctx, config, AuthOptions{})
}

func fetchTokenFromWebWithBrowserOptions(ctx context.Context, config *oauth2.Config, opts AuthOptions) (*oauth2.Token, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local OAuth callback listener: %w", err)
	}
	defer ln.Close()

	state, err := generateOAuthState()
	if err != nil {
		return nil, err
	}
	verifier := oauth2.GenerateVerifier()

	cfgCopy := *config
	cfgCopy.RedirectURL = fmt.Sprintf("http://%s", ln.Addr().String())
	authURL := buildAuthURL(&cfgCopy, state, verifier, opts.PromptConsent, opts.IncludeGrantedScopes)

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	srv := &http.Server{
		Handler:           newOAuthCallbackHandler(state, codeChan, errChan),
		ReadHeaderTimeout: 10 * time.Second,
	}
	defer func() {
		_ = srv.Shutdown(context.WithoutCancel(ctx))
	}()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sendNonBlockingErr(errChan, err)
		}
	}()

	fmt.Fprintln(os.Stderr, "Your browser has been opened to visit:", authURL)
	if err := browser.OpenURL(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case code := <-codeChan:
		return cfgCopy.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	case err := <-errChan:
		return nil, fmt.Errorf("OAuth callback failed: %w", err)
	}
}

// FetchTokenFromWeb requests an OAuth 2.0 token from the web via manual terminal code entry using PKCE (S256).
func FetchTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	return fetchTokenFromWebOptions(ctx, config, AuthOptions{})
}

func fetchTokenFromWebOptions(ctx context.Context, config *oauth2.Config, opts AuthOptions) (*oauth2.Token, error) {
	state, err := generateOAuthState()
	if err != nil {
		return nil, err
	}
	verifier := oauth2.GenerateVerifier()

	authURL := buildAuthURL(config, state, verifier, opts.PromptConsent, opts.IncludeGrantedScopes)
	fmt.Fprintf(os.Stderr, "Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := config.Exchange(ctx, authCode, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

// TokenCacheFile returns the path to the default profile's token cache file.
func TokenCacheFile() (string, error) {
	return ProfileTokenCacheFile(DefaultProfile)
}

// TokenFromFile retrieves a cached OAuth 2.0 token from a file, transparently decrypting
// AES-256-GCM envelopes and remaining backwards-compatible with legacy plaintext token files.
func TokenFromFile(file string) (*oauth2.Token, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var env encryptedTokenEnvelope
	if err := json.Unmarshal(raw, &env); err == nil && env.Format == encryptedTokenFormat {
		key, err := loadOrCreateEncryptionKey(filepath.Dir(file))
		if err != nil {
			return nil, err
		}
		raw, err = decryptTokenEnvelope(env, key)
		if err != nil {
			return nil, err
		}
	}

	tok := &oauth2.Token{}
	if err := json.Unmarshal(raw, tok); err != nil {
		return nil, fmt.Errorf("parsing cached token %q: %w", file, err)
	}
	return tok, nil
}

// SaveToken encrypts an OAuth 2.0 token with AES-256-GCM and writes it atomically with 0600 permissions.
func SaveToken(file string, token *oauth2.Token) error {
	fmt.Fprintf(os.Stderr, "Saving encrypted credential file to: %s\n", file)
	plaintext, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encoding oauth token: %w", err)
	}

	key, err := loadOrCreateEncryptionKey(filepath.Dir(file))
	if err != nil {
		return fmt.Errorf("loading token encryption key: %w", err)
	}

	encrypted, err := encryptTokenBytes(plaintext, key)
	if err != nil {
		return fmt.Errorf("encrypting oauth token: %w", err)
	}

	if err := writeFileAtomic(file, append(encrypted, '\n'), 0o600); err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	return nil
}

func fetchOAuthTokenOptions(ctx context.Context, config *oauth2.Config, opts AuthOptions) (*oauth2.Token, error) {
	if opts.NoBrowserAuth {
		return fetchTokenFromWebOptions(ctx, config, opts)
	}
	return fetchTokenFromWebWithBrowserOptions(ctx, config, opts)
}

func generateOAuthState() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generating OAuth CSRF state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func newOAuthCallbackHandler(expectedState string, codeChan chan<- string, errChan chan<- error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
			http.Error(w, "Authorization denied.", http.StatusBadRequest)
			sendNonBlockingErr(errChan, fmt.Errorf("authorization error from Google: %s", oauthErr))
			return
		}

		gotState := r.URL.Query().Get("state")
		if expectedState == "" || subtle.ConstantTimeCompare([]byte(gotState), []byte(expectedState)) != 1 {
			http.Error(w, "Invalid OAuth state parameter.", http.StatusBadRequest)
			sendNonBlockingErr(errChan, errors.New("OAuth state mismatch (possible CSRF)"))
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Invalid request. No authorization code received.", http.StatusBadRequest)
			sendNonBlockingErr(errChan, errors.New("no code in request"))
			return
		}

		fmt.Fprintln(w, "Authentication successful! You can close this browser window.")
		select {
		case codeChan <- code:
		default:
		}
	})
	return mux
}

func sendNonBlockingErr(errChan chan<- error, err error) {
	select {
	case errChan <- err:
	default:
	}
}
