package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/illmade-knight/action-intention/app"
	"github.com/illmade-knight/action-intention/pkg/crypto"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/action-intention/pkg/people"
	"github.com/illmade-knight/go-secure-messaging/pkg/transport"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Dependencies ---

type mockKeyClient struct {
	GetKeyFunc func(ctx context.Context, userID string) ([]byte, error)
}

func (m *mockKeyClient) GetKey(ctx context.Context, userID string) ([]byte, error) {
	return m.GetKeyFunc(ctx, userID)
}

type mockRouteClient struct {
	SendFunc func(ctx context.Context, envelope *transport.SecureEnvelope) error
}

func (m *mockRouteClient) Send(ctx context.Context, envelope *transport.SecureEnvelope) error {
	return m.SendFunc(ctx, envelope)
}

// --- Test Suite ---

func TestApp_ShareIntention(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	// Arrange: Generate cryptographic keys for sender and recipient
	senderPrivKey, _, err := crypto.GenerateKeys()
	require.NoError(t, err)
	_, recipientPubKey, err := crypto.GenerateKeys()
	require.NoError(t, err)

	// Arrange: Create in-memory stores and services with test data
	locStore := locations.NewInMemoryStore()
	locSvc := locations.NewService(locStore)
	testLoc, _ := locSvc.AddSharedLocation(ctx, "Test Park", "Recreation")

	personStore := people.NewInMemoryStore()
	personSvc := people.NewService(personStore)

	intentionStore := intentions.NewInMemoryStore()
	intentionSvc := intentions.NewIntentionService(intentionStore)
	testIntent, _ := intentionSvc.AddIntention(ctx, "sender", "Meet", []intentions.Target{
		intentions.LocationTarget{LocationID: testLoc.ID},
	}, time.Now(), time.Now().Add(1*time.Hour))

	// Arrange: Create mock clients
	keyClient := &mockKeyClient{
		GetKeyFunc: func(ctx context.Context, userID string) ([]byte, error) {
			assert.Equal(t, "recipient", userID)
			return recipientPubKey, nil
		},
	}

	sendCalled := false
	routeClient := &mockRouteClient{
		SendFunc: func(ctx context.Context, envelope *transport.SecureEnvelope) error {
			sendCalled = true
			assert.Equal(t, "sender", envelope.SenderID)
			assert.Equal(t, "recipient", envelope.RecipientID)
			require.NotEmpty(t, envelope.EncryptedData)
			require.NotEmpty(t, envelope.Signature)
			return nil
		},
	}

	// Arrange: Create the App instance with all dependencies
	application := app.New(intentionSvc, locSvc, personSvc, keyClient, routeClient, logger)

	// Act: Call the main workflow method
	err = application.ShareIntention(ctx, "sender", "recipient", testIntent.ID, senderPrivKey)

	// Assert
	require.NoError(t, err)
	assert.True(t, sendCalled, "Expected the routing service client's Send method to be called")
}
