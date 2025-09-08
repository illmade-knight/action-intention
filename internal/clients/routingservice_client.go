// Package clients provides HTTP clients for communicating with external microservices.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/illmade-knight/go-secure-messaging/pkg/transport"
	"github.com/rs/zerolog"
)

// RoutingServiceClient is responsible for all communication with the go-routing-service.
type RoutingServiceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewRoutingServiceClient creates a new client for the routing service.
func NewRoutingServiceClient(baseURL string, logger zerolog.Logger) *RoutingServiceClient {
	return &RoutingServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger.With().Str("client", "routing-service").Logger(),
	}
}

// Send dispatches a SecureEnvelope to the routing service for delivery.
func (c *RoutingServiceClient) Send(ctx context.Context, envelope *transport.SecureEnvelope) error {
	url := c.baseURL + "/send"

	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal secure envelope: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create send envelope request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute send envelope request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("routing service returned unexpected status code: %d", resp.StatusCode)
	}

	c.logger.Info().Str("sender_id", envelope.SenderID).Str("recipient_id", envelope.RecipientID).Msg("Successfully sent envelope to routing service")
	return nil
}
