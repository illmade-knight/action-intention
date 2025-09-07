//go:build integration

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/google/uuid"
	"github.com/illmade-knight/go-dataflow/pkg/cache"
	"github.com/illmade-knight/go-secure-messaging/pkg/transport"
	"github.com/illmade-knight/go-test/emulators"
	"github.com/illmade-knight/routing-service/pkg/routing"
	"github.com/illmade-knight/routing-service/routingservice"
	"github.com/illmade-knight/routing-service/test" // Import the new helpers
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPushNotifier simulates sending a notification to an offline user.
type mockPushNotifier struct {
	handled chan string
}

func (m *mockPushNotifier) Notify(ctx context.Context, tokens []routing.DeviceToken, envelope *transport.SecureEnvelope) error {
	m.handled <- envelope.RecipientID
	return nil
}

// mockDeliveryProducer implements routing.DeliveryProducer for the E2E test.
// It uses a real Pub/Sub client to publish to a delivery topic within our emulator.
type mockDeliveryProducer struct {
	pubsubClient *pubsub.Client
	logger       zerolog.Logger
}

// mockPresenceSeeder implements the cache.Fetcher interface to seed the presence cache.
type mockPresenceSeeder struct {
	presenceData map[string]routing.ConnectionInfo
}

// Fetch implements the cache.Fetcher interface.
func (m *mockPresenceSeeder) Fetch(ctx context.Context, key string) (routing.ConnectionInfo, error) {
	if val, ok := m.presenceData[key]; ok {
		return val, nil
	}
	return routing.ConnectionInfo{}, errors.New("not found in seed data")
}

// Close implements the cache.Fetcher interface (io.Closer).
func (m *mockPresenceSeeder) Close() error { return nil }

func (m *mockDeliveryProducer) Publish(ctx context.Context, topicID string, data *transport.SecureEnvelope) error {
	topic := m.pubsubClient.Publisher(topicID)
	defer topic.Stop()
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	result := topic.Publish(ctx, &pubsub.Message{Data: payload})
	_, err = result.Get(ctx)
	if err != nil {
		m.logger.Error().Err(err).Str("topic", topicID).Msg("E2E mockDeliveryProducer failed to publish")
	}
	return err
}

// mockDeviceTokenSeeder implements the cache.Fetcher interface to seed the device token cache.
type mockDeviceTokenSeeder struct {
	tokenData map[string][]routing.DeviceToken
}

// Fetch implements the cache.Fetcher interface.
func (m *mockDeviceTokenSeeder) Fetch(ctx context.Context, key string) ([]routing.DeviceToken, error) {
	if val, ok := m.tokenData[key]; ok {
		return val, nil
	}
	return nil, errors.New("not found in seed data")
}

// Close implements the cache.Fetcher interface (io.Closer).
func (m *mockDeviceTokenSeeder) Close() error { return nil }

func TestRoutingService_E2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	const projectID = "test-project"
	runID := uuid.NewString()

	// 1. Setup Emulators and Pub/Sub Client
	pubsubConn := emulators.SetupPubsubEmulator(t, ctx, emulators.GetDefaultPubsubConfig(projectID))
	psClient, err := pubsub.NewClient(ctx, projectID, pubsubConn.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = psClient.Close() })

	// 2. Create Pub/Sub Resources
	ingressTopicID := "ingress-topic-" + runID
	ingressSubID := "ingress-sub-" + runID
	deliveryTopicID := "delivery-pod-123"
	deliverySubID := "delivery-sub-" + runID
	createPubsubResources(t, ctx, psClient, projectID, ingressTopicID, ingressSubID)
	createPubsubResources(t, ctx, psClient, projectID, deliveryTopicID, deliverySubID)

	// 3. Setup Test Dependencies (Mocks and Caches)
	// Create a mock fetcher (seeder) with the seed data for our online user.
	presenceSeeder := &mockPresenceSeeder{
		presenceData: map[string]routing.ConnectionInfo{
			"user-online": {ServerInstanceID: "pod-123"},
		},
	}
	// Create the InMemoryCache with the seeder as its fallback.
	presenceCache := cache.NewInMemoryCache[string, routing.ConnectionInfo](presenceSeeder)

	// To seed the cache, we perform an initial fetch. This is a cache miss,
	// which triggers the fallback seeder and populates the cache.
	_, err = presenceCache.Fetch(ctx, "user-online")
	require.NoError(t, err, "Failed to seed presence cache via fallback")

	// Create a seeder with mock device tokens for the offline user.
	tokenSeeder := &mockDeviceTokenSeeder{
		tokenData: map[string][]routing.DeviceToken{
			"user-offline": {{Token: "mock-device-token-abc"}},
		},
	}
	// Create the cache with the seeder as its fallback.
	deviceTokenFetcher := cache.NewInMemoryCache[string, []routing.DeviceToken](tokenSeeder)

	offlineHandled := make(chan string, 1)
	deps := &routing.Dependencies{
		PresenceCache:      presenceCache,
		DeviceTokenFetcher: deviceTokenFetcher, // Use the newly created fetcher
		PushNotifier:       &mockPushNotifier{handled: offlineHandled},
		DeliveryProducer:   &mockDeliveryProducer{pubsubClient: psClient, logger: logger},
	}

	// 4. Instantiate Concrete Adapters using the new TEST HELPERS
	consumer, err := test.NewTestConsumer(ingressSubID, psClient, logger)
	require.NoError(t, err)

	ingestionTopic := psClient.Publisher(ingressTopicID)
	defer ingestionTopic.Stop()
	ingestionProducer := test.NewTestProducer(ingestionTopic)

	// 5. Configure and Start the Service using the public constructor
	cfg := &routing.Config{
		HTTPListenAddr:     ":0", // Use a random port
		NumPipelineWorkers: 5,
	}
	service, err := routingservice.New(cfg, deps, consumer, ingestionProducer, logger)
	require.NoError(t, err)

	err = service.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = service.Shutdown(shutdownCtx)
	})

	// 6. Verification Goroutines
	var wg sync.WaitGroup
	wg.Add(2)
	go verifyOnlineUser(t, ctx, &wg, psClient.Subscriber(deliverySubID))
	go verifyOfflineUser(t, ctx, &wg, offlineHandled)

	// 7. Send Test Requests
	var serviceURL string
	require.Eventually(t, func() bool {
		port := service.GetHTTPPort()
		if port == "" {
			return false
		}
		serviceURL = "http://localhost" + port
		return true
	}, 5*time.Second, 50*time.Millisecond, "HTTP server never started listening")

	sendRequest(t, "user-online", serviceURL+"/send")
	sendRequest(t, "user-offline", serviceURL+"/send")

	wg.Wait()
}

// --- E2E Test Helpers ---

func createPubsubResources(t *testing.T, ctx context.Context, client *pubsub.Client, projectID, topicID, subID string) {
	t.Helper()
	topicAdminClient := client.TopicAdminClient
	subAdminClient := client.SubscriptionAdminClient

	topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)
	_, err := topicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = topicAdminClient.DeleteTopic(context.Background(), &pubsubpb.DeleteTopicRequest{Topic: topicName})
	})

	subName := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subID)
	_, err = subAdminClient.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:  subName,
		Topic: topicName,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = subAdminClient.DeleteSubscription(context.Background(), &pubsubpb.DeleteSubscriptionRequest{Subscription: subName})
	})
}

func sendRequest(t *testing.T, recipientID, url string) {
	t.Helper()
	envelope := transport.SecureEnvelope{
		SenderID:    "e2e-sender",
		RecipientID: recipientID,
	}
	body, err := json.Marshal(envelope)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func verifyOnlineUser(t *testing.T, ctx context.Context, wg *sync.WaitGroup, sub *pubsub.Subscriber) {
	defer wg.Done()
	verifyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := sub.Receive(verifyCtx, func(ctxRec context.Context, msg *pubsub.Message) {
		msg.Ack()
		var envelope transport.SecureEnvelope
		err := json.Unmarshal(msg.Data, &envelope)
		require.NoError(t, err)
		if envelope.RecipientID == "user-online" {
			t.Log("✅ SUCCESS: Correctly received message for ONLINE user.")
			cancel() // Stop receiving
		}
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Receiving from delivery subscription failed: %v", err)
	}
}

func verifyOfflineUser(t *testing.T, ctx context.Context, wg *sync.WaitGroup, handled chan string) {
	defer wg.Done()
	select {
	case recipient := <-handled:
		assert.Equal(t, "user-offline", recipient)
		t.Log("✅ SUCCESS: Correctly handled OFFLINE user.")
	case <-ctx.Done():
		t.Error("Timeout waiting for offline user to be handled")
	}
}
