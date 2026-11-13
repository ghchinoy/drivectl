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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"
)

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

// NewOAuthClient creates a new HTTP client with OAuth 2.0 authentication.
func NewOAuthClient(ctx context.Context, secretFile string, noBrowserAuth bool) (*http.Client, error) {
	if secretFile == "" {
		configDir, err := ConfigDir()
		if err != nil {
			return nil, err
		}
		secretFile = filepath.Join(configDir, "client_secret.json")
	}

	b, err := os.ReadFile(secretFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file at %q: %w", secretFile, err)
	}

	config, err := google.ConfigFromJSON(
		b,
		drive.DriveReadonlyScope,
		docs.DocumentsReadonlyScope,
		sheets.SpreadsheetsScope,
		slides.PresentationsReadonlyScope,
		drive.DriveFileScope,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	return FetchClient(ctx, config, noBrowserAuth)
}

// FetchClient retrieves a token from a local file or the web, then returns an authenticated HTTP client.
func FetchClient(ctx context.Context, config *oauth2.Config, noBrowserAuth bool) (*http.Client, error) {
	cacheFile, err := TokenCacheFile()
	if err != nil {
		return nil, fmt.Errorf("unable to get path to cached credential file: %w", err)
	}

	tok, err := TokenFromFile(cacheFile)
	if err != nil {
		tok, err = fetchOAuthToken(ctx, config, noBrowserAuth)
		if err != nil {
			return nil, err
		}
		if err := SaveToken(cacheFile, tok); err != nil {
			return nil, err
		}
	}
	return config.Client(ctx, tok), nil
}

func fetchOAuthToken(ctx context.Context, config *oauth2.Config, noBrowserAuth bool) (*oauth2.Token, error) {
	if noBrowserAuth {
		return FetchTokenFromWeb(ctx, config)
	}
	return FetchTokenFromWebWithBrowser(ctx, config)
}

// FetchTokenFromWebWithBrowser uses a local web server to handle the OAuth 2.0 callback flow.
func FetchTokenFromWebWithBrowser(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = "http://localhost:8080"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Println("Your browser has been opened to visit:", authURL)
	if err := browser.OpenURL(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: newOAuthCallbackHandler(codeChan, errChan),
	}
	defer func() {
		_ = srv.Shutdown(ctx)
	}()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sendNonBlockingErr(errChan, err)
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case code := <-codeChan:
		return config.Exchange(ctx, code)
	case err := <-errChan:
		return nil, fmt.Errorf("failed to start local server or get code: %w", err)
	}
}

func newOAuthCallbackHandler(codeChan chan<- string, errChan chan<- error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintln(w, "Invalid request. No authorization code received.")
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

// FetchTokenFromWeb requests an OAuth 2.0 token from the web via manual terminal code entry.
func FetchTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

// TokenCacheFile returns the path to the token cache file.
func TokenCacheFile() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "token.json"), nil
}

// TokenFromFile retrieves a cached OAuth 2.0 token from a file.
func TokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// SaveToken saves an OAuth 2.0 token to a file with restricted permissions.
func SaveToken(file string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	err = json.NewEncoder(f).Encode(token)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}
