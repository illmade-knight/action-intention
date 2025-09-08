package clients_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/illmade-knight/action-intention/internal/clients"
	"github.com/illmade-knight/go-secure-messaging/pkg/transport"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutingServiceClient_Send(t *testing.T) {
	ctx := context.Background()
	testEnvelope := &transport.SecureEnvelope{
		SenderID:    "user-alice",
		RecipientID: "user-bob",
	}

	// Arrange: Create a mock HTTP server to act as the routing service
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/send", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var receivedEnvelope transport.SecureEnvelope
		err := json.NewDecoder(r.Body).Decode(&receivedEnvelope)
		require.NoError(t, err)
		assert.Equal(t, testEnvelope.RecipientID, receivedEnvelope.RecipientID)

		w.WriteHeader(http.StatusAccepted)
	}))
	defer mockServer.Close()

	// Arrange: Create the client pointing to our mock server
	client := clients.NewRoutingServiceClient(mockServer.URL, zerolog.Nop())

	// Act
	err := client.Send(ctx, testEnvelope)

	// Assert
	require.NoError(t, err)
}
