package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/firestore"
	"github.com/illmade-knight/action-intention/app"
	"github.com/illmade-knight/action-intention/internal/clients"
	firestorestorage "github.com/illmade-knight/action-intention/internal/storage/firestore"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/action-intention/pkg/people"
	"github.com/rs/zerolog"
)

// Config holds the application's configuration. In a real application,
// this would be populated from a file or environment variables.
type Config struct {
	GCPProjectID      string
	KeyServiceURL     string
	RoutingServiceURL string
}

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. Load Configuration
	cfg := Config{
		GCPProjectID:      os.Getenv("GCP_PROJECT_ID"),
		KeyServiceURL:     "http://localhost:8081", // Example URL
		RoutingServiceURL: "http://localhost:8080", // Example URL
	}
	if cfg.GCPProjectID == "" {
		logger.Fatal().Msg("GCP_PROJECT_ID environment variable must be set.")
	}

	// 2. Initialize External Clients (e.g., Firestore)
	fsClient, err := firestore.NewClient(ctx, cfg.GCPProjectID)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create Firestore client")
	}
	defer fsClient.Close()

	// 3. Instantiate Persistent Storage Adapters
	locationsStore := firestorestorage.NewLocationsStore(fsClient)
	peopleStore := firestorestorage.NewPeopleStore(fsClient)
	intentionsStore := firestorestorage.NewIntentionsStore(fsClient)
	logger.Info().Msg("Firestore storage adapters initialized")

	// 4. Instantiate Domain Services
	locationSvc := locations.NewService(locationsStore)
	personSvc := people.NewService(peopleStore)
	intentionSvc := intentions.NewIntentionService(intentionsStore)
	logger.Info().Msg("Domain services initialized")

	// 5. Instantiate Networking Clients
	keyClient := clients.NewKeyServiceClient(cfg.KeyServiceURL, logger)
	routeClient := clients.NewRoutingServiceClient(cfg.RoutingServiceURL, logger)
	logger.Info().Msg("Networking clients initialized")

	// 6. Instantiate the Main Application Orchestrator
	application := app.New(intentionSvc, locationSvc, personSvc, keyClient, routeClient, logger)
	logger.Info().Str("app_address", fmt.Sprintf("%p", application)).Msg("Application orchestrator created")

	// --- Application is now fully assembled and ready ---
	logger.Info().Msg("Action-Intention service initialized. Waiting for shutdown signal...")
	// In Phase 2, this is where we would start the HTTP API server.
	// For now, the application will idle until it's terminated.

	<-ctx.Done()
	logger.Info().Msg("Shutdown signal received. Exiting.")
}
