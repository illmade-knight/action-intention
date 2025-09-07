// REFACTOR: This is the final, correct demo. It demonstrates how to use the
// public service wrapper from /pkg/routing for in-process, end-to-end testing.
// It correctly imports dependencies ONLY from public packages, resolving all
// previous import cycle and visibility issues.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	//"fmt"
	"log"
	"net/http"
	"time"

	"github.com/illmade-knight/go-secure-messaging/pkg/transport"
	"github.com/illmade-knight/routing-service/pkg/routing"
)

// --- Mocks for the Demo ---

// mockDeliveryProducer simulates forwarding a message to an online user.
// It implements the public routing.DeliveryProducer interface.
type mockDeliveryProducer struct {
	handled chan string
}

func (m *mockDeliveryProducer) Publish(ctx context.Context, topicID string, data *transport.SecureEnvelope) error {
	log.Printf("DEMO: [DeliveryProducer] successfully routed message to topic '%s' for user '%s'", topicID, data.RecipientID)
	m.handled <- "online"
	return nil
}

// mockPushNotifier simulates sending a notification to an offline user.
// It implements the public routing.PushNotifier interface.
type mockPushNotifier struct {
	handled chan string
}

func (m *mockPushNotifier) Notify(ctx context.Context, tokens []routing.DeviceToken, envelope *transport.SecureEnvelope) error {
	log.Printf("DEMO: [PushNotifier] successfully sent push notification to user '%s'", envelope.RecipientID)
	m.handled <- "offline"
	return nil
}

//TODO rewrite a demo for the full system once we've got a full key-service implementation as well

//func main() {
//	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
//	logger.Info().Msg("--- Starting In-Process Demo with Service Wrapper ---")
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	// --- 1. Create In-Memory Dependencies for the Demo ---
//	// This demonstrates the flexibility of the interface-based wrapper.
//	// We use the generic InMemoryCache from go-dataflow with our public types from /pkg/routing.
//	presenceCache := cache.NewInMemoryCache[string, routing.ConnectionInfo](nil)
//	deviceTokenFetcher := cache.NewInMemoryCache[string, []routing.DeviceToken](nil)
//
//	deliveryHandled := make(chan string, 1)
//	deliveryProducer := &mockDeliveryProducer{handled: deliveryHandled}
//	pushNotifier := &mockPushNotifier{handled: deliveryHandled}
//
//	// Seed the presence cache: user "user-online" is routingected.
//	// We use a type assertion to access the Write method for this in-memory implementation.
//	if writableCache, ok := presenceCache.(cache.Cache[string, routing.ConnectionInfo]); ok {
//		_ = writableCache.Write(ctx, "user-online", routing.ConnectionInfo{ServerInstanceID: "pod-123"})
//	}
//
//	// --- 2. Configure and Create the Service Wrapper ---
//	// The configuration and dependencies use the public structs from /pkg/routing.
//	cfg := &routing.Config{
//		HTTPListenAddr:        ":8081",
//		IngressSubscriptionID: "demo-sub",
//		IngressTopicID:        "demo-topic",
//		NumPipelineWorkers:    2,
//	}
//
//	// For a fully in-process demo, we simulate Pub/Sub in memory.
//	localPubSub := messagepipeline.NewInMemoryPubSub()
//	deps := &routing.Dependencies{
//		Logger:             logger,
//		PubsubClient:       localPubSub.Client,
//		PresenceCache:      presenceCache,
//		DeviceTokenFetcher: deviceTokenFetcher,
//		DeliveryProducer:   deliveryProducer,
//		PushNotifier:       pushNotifier,
//	}
//	cfg.PubsubClientOptions = []option.ClientOption{option.WithEndpoint("inmemory")}
//
//	// We instantiate the wrapper from the public /pkg/routing package.
//	service, err := routing.NewRoutingServiceWrapper(cfg, deps)
//	if err != nil {
//		logger.Fatal().Err(err).Msg("Failed to create routing service wrapper")
//	}
//
//	// --- 3. Start the Service and Run Test Cases ---
//	err = service.Start(ctx)
//	if err != nil {
//		logger.Fatal().Err(err).Msg("Failed to start service")
//	}
//
//	// Test Case 1: Send to an online user
//	log.Println("\n--- DEMO: Sending message to ONLINE user 'user-online' ---")
//	sendRequest("user-online", "http://localhost:8081/send")
//	assertDelivery(deliveryHandled, "online")
//
//	// Test Case 2: Send to an offline user
//	log.Println("\n--- DEMO: Sending message to OFFLINE user 'user-offline' ---")
//	sendRequest("user-offline", "http://localhost:8081/send")
//	assertDelivery(deliveryHandled, "offline")
//
//	// --- 4. Handle Graceful Shutdown ---
//	log.Println("\n--- DEMO: Shutting down ---")
//	stopChan := make(chan os.Signal, 1)
//	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
//	<-stopChan
//
//	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer shutdownCancel()
//	if err := service.Shutdown(shutdownCtx); err != nil {
//		logger.Error().Err(err).Msg("Service shutdown failed.")
//	}
//}

// --- Demo Helper Functions ---

func sendRequest(recipientID, url string) {
	envelope := transport.SecureEnvelope{
		SenderID:    "demo-sender",
		RecipientID: recipientID,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		log.Fatalf("Demo helper failed to marshal envelope: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("HTTP POST request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		log.Fatalf("Expected status 202 Accepted, got %d", resp.StatusCode)
	}
}

func assertDelivery(deliveryHandled chan string, expected string) {
	select {
	case result := <-deliveryHandled:
		if result != expected {
			log.Fatalf("Incorrect delivery path taken. Expected '%s', got '%s'", expected, result)
		}
		log.Printf("✅ SUCCESS: Message correctly routed via '%s' path.", expected)
	case <-time.After(3 * time.Second):
		log.Fatalf("Timeout: message was not processed by any delivery handler in time.")
	}
}
