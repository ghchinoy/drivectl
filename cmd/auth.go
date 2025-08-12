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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
)

// newOAuthClient creates a new HTTP client with OAuth 2.0 authentication.
func newOAuthClient(ctx context.Context, secretFile string, noBrowserAuth bool) (*http.Client, error) {
	b, err := os.ReadFile(secretFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope, docs.DocumentsReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	return getClient(ctx, config, noBrowserAuth)
}

// getClient retrieves a token from a local file or the web, then returns a client.
func getClient(ctx context.Context, config *oauth2.Config, noBrowserAuth bool) (*http.Client, error) {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		return nil, fmt.Errorf("unable to get path to cached credential file: %v", err)
	}

	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		if noBrowserAuth {
			tok, err = getTokenFromWeb(config)
		} else {
			tok, err = getTokenFromWebWithBrowser(config)
		}
		if err != nil {
			return nil, err
		}
		if err := saveToken(cacheFile, tok); err != nil {
			return nil, err
		}
	}
	return config.Client(ctx, tok), nil
}

// getTokenFromWebWithBrowser uses a local web server to handle the OAuth flow.
func getTokenFromWebWithBrowser(config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = "http://localhost:8080"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Println("Your browser has been opened to visit:", authURL)
	if err := browser.OpenURL(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	codeChan := make(chan string)
	errChan := make(chan error)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintln(w, "Invalid request. No authorization code received.")
			errChan <- fmt.Errorf("no code in request")
			return
		}
		fmt.Fprintln(w, "Authentication successful! You can close this browser window.")
		codeChan <- code
	})

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case code := <-codeChan:
		return config.Exchange(context.Background(), code)
	case err := <-errChan:
		return nil, fmt.Errorf("failed to start local server or get code: %w", err)
	}
}

// getTokenFromWeb requests a token from the web.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}
	return tok, nil
}

// tokenCacheFile returns the path to the token cache file.
func tokenCacheFile() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	tokenCacheDir := filepath.Join(homeDir, ".config", "drivectl")
	if err := os.MkdirAll(tokenCacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create token cache directory: %w", err)
	}
	return filepath.Join(tokenCacheDir, "token.json"), nil
}

// tokenFromFile retrieves a token from a file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file.
func saveToken(file string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
