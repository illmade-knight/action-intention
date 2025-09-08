//go:build integration

package e2e_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/illmade-knight/go-key-service/test" // Correctly import the test helper
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyService_E2E(t *testing.T) {
	// 1. Create and start a test server using the public helper
	server := test.NewTestServer()
	defer server.Close()

	// --- 2. Run Test Cases against the live server ---
	const testUserID = "e2e-user-1"
	const testKey = "my-e2e-public-key"
	keyURL := server.URL + "/keys/" + testUserID

	// --- Test POST (Store the key) ---
	t.Run("POST should store the key and return 201 Created", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, keyURL, bytes.NewReader([]byte(testKey)))
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	// --- Test GET (Retrieve the key) ---
	t.Run("GET should retrieve the previously stored key and return 200 OK", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, keyURL, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte(testKey), body)
	})
}
