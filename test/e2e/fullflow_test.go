//go:build integration

package e2e_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/pubsub/v2"
	"github.com/google/uuid"
	"github.com/illmade-knight/go-dataflow/pkg/cache"
	"github.com/illmade-knight/go-test/emulators"

	// Import the PUBLIC test helpers for BOTH services
	keytest "github.com/illmade-knight/go-key-service/test"
	routingtest "github.com/illmade-knight/routing-service/test"

	// Import the PUBLIC packages for routing service dependencies
	"github.com/illmade-knight/routing-service/pkg/routing"
	"github.com/illmade-knight/routing-service/routingservice"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestFullApplicationFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	logger := zerolog.New(zerolog.NewTestWriter(t))
	const projectID = "test-project"
	runID := uuid.NewString()

	// 1. SETUP: Start Emulators
	pubsubConn := emulators.SetupPubsubEmulator(t, ctx, emulators.GetDefaultPubsubConfig(projectID))
	psClient, err := pubsub.NewClient(ctx, projectID, pubsubConn.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = psClient.Close() })

	firestoreConn := emulators.SetupFirestoreEmulator(t, ctx, emulators.GetDefaultFirestoreConfig(projectID))
	fsClient, err := firestore.NewClient(ctx, projectID, firestoreConn.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = fsClient.Close() })

	// 2. ARRANGE: Start the Key Service
	keyServer := keytest.NewTestKeyService(fsClient, "public-keys")
	defer keyServer.Close()

	// 3. ARRANGE: Start the Routing Service
	targetUser := "user-with-persistent-device-token"

	// Pre-populate Firestore with data
	deviceDocRef := fsClient.Collection("device-tokens").Doc(targetUser)
	_, err = deviceDocRef.Set(ctx, map[string]interface{}{
		"Tokens": []map[string]interface{}{
			{"Token": "persistent-device-token-123", "Platform": "ios"},
		},
	})
	require.NoError(t, err)

	// Create the Firestore token fetcher using our new test helper
	tokenFetcher, err := routingtest.NewTestFirestoreTokenFetcher(ctx, fsClient, projectID, logger)
	require.NoError(t, err)

	offlineHandled := make(chan string, 1)
	routingDeps := &routing.Dependencies{
		PresenceCache:      cache.NewInMemoryCache[string, routing.ConnectionInfo](nil),
		DeviceTokenFetcher: tokenFetcher,
		PushNotifier:       &mockPushNotifier{handled: offlineHandled},
	}

	ingressTopicID := "ingress-topic-" + runID
	createPubsubResources(t, ctx, psClient, projectID, ingressTopicID, "sub-"+runID)
	consumer, _ := routingtest.NewTestConsumer("sub-"+runID, psClient, logger)
	producer := routingtest.NewTestProducer(psClient.Publisher(ingressTopicID))

	routingService, err := routingservice.New(&routing.Config{HTTPListenAddr: ":0"}, routingDeps, consumer, producer, logger)
	require.NoError(t, err)
	require.NoError(t, routingService.Start(ctx))
	t.Cleanup(func() { _ = routingService.Shutdown(context.Background()) })

	// 4. ACT: Execute the test scenario
	// Step A: Client registers their public key.
	keyURL := keyServer.URL + "/keys/" + targetUser
	req, _ := http.NewRequest(http.MethodPost, keyURL, bytes.NewReader([]byte("my-new-public-key")))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Step B: Another client sends a message to the target user.
	var routingServerURL string
	require.Eventually(t, func() bool {
		port := routingService.GetHTTPPort()
		if port != "" {
			routingServerURL = "http://localhost" + port
			return true
		}
		return false
	}, 5*time.Second, 50*time.Millisecond)
	sendRequest(t, targetUser, routingServerURL+"/send")

	// 5. ASSERT: Verify the outcome
	select {
	case recipient := <-offlineHandled:
		require.Equal(t, targetUser, recipient)
		t.Log("✅ SUCCESS: Full flow complete. Routing service fetched device token from Firestore and triggered push notification.")
	case <-time.After(15 * time.Second):
		t.Fatal("Test timed out: routing service did not trigger the push notifier")
	}
}
