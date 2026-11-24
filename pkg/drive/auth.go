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
	"os"
	"path/filepath"
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
	profile, err := NormalizeProfile(opts.Profile)
	if err != nil {
		return nil, err
	}

	secretFile := opts.SecretFile
	if secretFile == "" {
		secretFile, err = ProfileSecretFile(profile)
		if err != nil {
			return nil, err
		}
	}

	b, err := os.ReadFile(secretFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file at %q: %w", secretFile, err)
	}

	config, err := google.ConfigFromJSON(b, DefaultOAuthScopes()...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	cacheFile, err := ProfileTokenCacheFile(profile)
	if err != nil {
		return nil, fmt.Errorf("unable to get path to cached credential file: %w", err)
	}

	client, err := fetchClientWithCache(ctx, config, cacheFile, opts.NoBrowserAuth)
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
	tok, err := TokenFromFile(cacheFile)
	if err != nil {
		tok, err = fetchOAuthToken(ctx, config, noBrowserAuth)
		if err != nil {
			return nil, err
		}
		if err := SaveToken(cacheFile, tok); err != nil {
			return nil, err
		}
	} else if !InspectTokenFile(cacheFile).Encrypted {
		// Upgrade legacy plaintext token cache in place to AES-256-GCM.
		_ = SaveToken(cacheFile, tok)
	}

	ts := newPersistingTokenSource(config.TokenSource(ctx, tok), cacheFile, tok)
	return oauth2.NewClient(ctx, ts), nil
}

// FetchTokenFromWebWithBrowser uses an ephemeral loopback web server with PKCE (S256)
// and CSRF state verification to handle the OAuth 2.0 callback flow.
func FetchTokenFromWebWithBrowser(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
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
	authURL := cfgCopy.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)

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
	state, err := generateOAuthState()
	if err != nil {
		return nil, err
	}
	verifier := oauth2.GenerateVerifier()

	authURL := config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)
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

func fetchOAuthToken(ctx context.Context, config *oauth2.Config, noBrowserAuth bool) (*oauth2.Token, error) {
	if noBrowserAuth {
		return FetchTokenFromWeb(ctx, config)
	}
	return FetchTokenFromWebWithBrowser(ctx, config)
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
