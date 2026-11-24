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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

const (
	encryptedTokenFormat = "drivectl-aes256gcm-v1"
	encryptionKeyEnvVar  = "DRIVECTL_ENCRYPTION_KEY"
	encryptionKeyFile    = ".encryption_key"
	aes256KeySize        = 32
)

type encryptedTokenEnvelope struct {
	Format     string `json:"format"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// TokenStoreMetadata describes how a cached token file is stored on disk.
type TokenStoreMetadata struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	Encrypted bool   `json:"encrypted"`
	KeySource string `json:"key_source,omitempty"`
}

// InspectTokenFile reports whether a token cache file exists and whether it is encrypted at rest.
func InspectTokenFile(file string) TokenStoreMetadata {
	meta := TokenStoreMetadata{Path: file}
	raw, err := os.ReadFile(file)
	if err != nil {
		return meta
	}
	meta.Exists = true

	var env encryptedTokenEnvelope
	if err := json.Unmarshal(raw, &env); err == nil && env.Format == encryptedTokenFormat {
		meta.Encrypted = true
		meta.KeySource = filepath.Join(filepath.Dir(file), encryptionKeyFile)
		if strings.TrimSpace(os.Getenv(encryptionKeyEnvVar)) != "" {
			meta.KeySource = "env:" + encryptionKeyEnvVar
		}
	}
	return meta
}

// loadOrCreateEncryptionKey resolves the 32-byte AES-256 key from DRIVECTL_ENCRYPTION_KEY
// or a 0600 key file in keyDir, generating one on first use.
func loadOrCreateEncryptionKey(keyDir string) ([]byte, error) {
	if envKey := strings.TrimSpace(os.Getenv(encryptionKeyEnvVar)); envKey != "" {
		return decodeAESKey(envKey, encryptionKeyEnvVar)
	}

	keyPath := filepath.Join(keyDir, encryptionKeyFile)
	raw, err := os.ReadFile(keyPath)
	if err == nil {
		return decodeAESKey(strings.TrimSpace(string(raw)), keyPath)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading encryption key %q: %w", keyPath, err)
	}

	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating key directory %q: %w", keyDir, err)
	}

	key := make([]byte, aes256KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generating encryption key: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(key) + "\n"
	if err := writeFileAtomic(keyPath, []byte(encoded), 0o600); err != nil {
		return nil, fmt.Errorf("writing encryption key %q: %w", keyPath, err)
	}
	return key, nil
}

func decodeAESKey(encoded, source string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding encryption key from %s: %w", source, err)
	}
	if len(key) != aes256KeySize {
		return nil, fmt.Errorf("invalid encryption key length from %s: got %d bytes, want %d", source, len(key), aes256KeySize)
	}
	return key, nil
}

func encryptTokenBytes(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initializing AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initializing GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating GCM nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, []byte(encryptedTokenFormat))
	env := encryptedTokenEnvelope{
		Format:     encryptedTokenFormat,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	return json.Marshal(env)
}

func decryptTokenEnvelope(env encryptedTokenEnvelope, key []byte) ([]byte, error) {
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decoding token nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decoding token ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initializing AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initializing GCM: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid token nonce length: got %d, want %d", len(nonce), gcm.NonceSize())
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(encryptedTokenFormat))
	if err != nil {
		return nil, fmt.Errorf("decrypting cached token: %w", err)
	}
	return plaintext, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".drivectl-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// persistingTokenSource wraps an oauth2.TokenSource and re-saves tokens whenever they refresh.
type persistingTokenSource struct {
	base      oauth2.TokenSource
	cacheFile string

	mu   sync.Mutex
	last *oauth2.Token
}

func newPersistingTokenSource(base oauth2.TokenSource, cacheFile string, initial *oauth2.Token) oauth2.TokenSource {
	return &persistingTokenSource{
		base:      base,
		cacheFile: cacheFile,
		last:      initial,
	}
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.base.Token()
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.last == nil || tok.AccessToken != p.last.AccessToken || tok.RefreshToken != p.last.RefreshToken {
		if err := SaveToken(p.cacheFile, tok); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to persist refreshed token: %v\n", err)
		}
		p.last = tok
	}
	return tok, nil
}
