// Package clients provides HTTP clients for communicating with external microservices.
package clients

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// KeyServiceClient is responsible for all communication with the go-key-service.
type KeyServiceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewKeyServiceClient creates a new client for the key service.
func NewKeyServiceClient(baseURL string, logger zerolog.Logger) *KeyServiceClient {
	return &KeyServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger.With().Str("client", "key-service").Logger(),
	}
}

// GetKey fetches a user's public key.
func (c *KeyServiceClient) GetKey(ctx context.Context, userID string) ([]byte, error) {
	url := fmt.Sprintf("%s/keys/%s", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get key request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get key request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("key for user %s not found", userID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("key service returned unexpected status code: %d", resp.StatusCode)
	}

	key, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read key from response body: %w", err)
	}

	c.logger.Info().Str("user_id", userID).Msg("Successfully fetched public key")
	return key, nil
}

// StoreKey uploads a user's public key.
func (c *KeyServiceClient) StoreKey(ctx context.Context, userID string, key []byte) error {
	url := fmt.Sprintf("%s/keys/%s", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(key))
	if err != nil {
		return fmt.Errorf("failed to create store key request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute store key request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("key service returned unexpected status code for store: %d", resp.StatusCode)
	}

	c.logger.Info().Str("user_id", userID).Msg("Successfully stored public key")
	return nil
}
