package clients_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/illmade-knight/action-intention/internal/clients"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyServiceClient(t *testing.T) {
	const testUserID = "user-123"
	const testKey = "my-public-key"
	ctx := context.Background()

	// Arrange: Create a mock HTTP server to act as the key service
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/keys/" + testUserID
		if r.URL.Path != expectedPath {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(testKey))
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer mockServer.Close()

	// Arrange: Create the client pointing to our mock server
	client := clients.NewKeyServiceClient(mockServer.URL, zerolog.Nop())

	t.Run("GetKey - Success", func(t *testing.T) {
		// Act
		key, err := client.GetKey(ctx, testUserID)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []byte(testKey), key)
	})

	t.Run("GetKey - Not Found", func(t *testing.T) {
		// Act
		_, err := client.GetKey(ctx, "non-existent-user")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("StoreKey - Success", func(t *testing.T) {
		// Act
		err := client.StoreKey(ctx, testUserID, []byte(testKey))

		// Assert
		require.NoError(t, err)
	})
}
