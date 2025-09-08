// Package app provides the central orchestrator for the action-intention application.
package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/crypto"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/action-intention/pkg/people"
	"github.com/illmade-knight/action-intention/pkg/reconciliation"
	"github.com/illmade-knight/action-intention/pkg/sharing"
	"github.com/illmade-knight/go-secure-messaging/pkg/transport"
	"github.com/rs/zerolog"
)

// KeyFetcher defines the interface for a component that can fetch a user's public key.
type KeyFetcher interface {
	GetKey(ctx context.Context, userID string) ([]byte, error)
}

// EnvelopeSender defines the interface for a component that can send a secure envelope.
type EnvelopeSender interface {
	Send(ctx context.Context, envelope *transport.SecureEnvelope) error
}

// App is the central application struct. It holds all domain services, clients,
// and other components needed to run the application's business logic.
type App struct {
	IntentionSvc *intentions.IntentionService
	LocationSvc  *locations.Service
	PersonSvc    *people.Service
	Reconciler   *reconciliation.Reconciler
	KeyClient    KeyFetcher
	RouteClient  EnvelopeSender
	Logger       zerolog.Logger
}

// New creates a new, fully initialized App.
func New(
	intentionSvc *intentions.IntentionService,
	locationSvc *locations.Service,
	personSvc *people.Service,
	keyClient KeyFetcher,
	routeClient EnvelopeSender,
	logger zerolog.Logger,
) *App {
	reconciler := reconciliation.NewReconciler(locationSvc.GetStore(), personSvc.GetStore())
	return &App{
		IntentionSvc: intentionSvc,
		LocationSvc:  locationSvc,
		PersonSvc:    personSvc,
		Reconciler:   reconciler,
		KeyClient:    keyClient,
		RouteClient:  routeClient,
		Logger:       logger,
	}
}

// ShareIntention orchestrates the entire process of securely sharing an intention.
func (a *App) ShareIntention(ctx context.Context, senderID, recipientID string, intentionID uuid.UUID, privateKeyPEM []byte) error {
	logger := a.Logger.With().
		Str("sender_id", senderID).
		Str("recipient_id", recipientID).
		Stringer("intention_id", intentionID).
		Logger()

	logger.Info().Msg("Beginning intention sharing workflow")

	// 1. Build the SharedPayload
	payload, err := a.buildSharedPayload(ctx, intentionID)
	if err != nil {
		return fmt.Errorf("failed to build shared payload: %w", err)
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal shared payload: %w", err)
	}

	// 2. Get the Recipient's Public Key
	recipientPubKey, err := a.KeyClient.GetKey(ctx, recipientID)
	if err != nil {
		return fmt.Errorf("failed to get recipient's public key: %w", err)
	}

	// 3. Encrypt and Sign the payload
	// The AAD (Additional Authenticated Data) includes sender and recipient IDs
	// to prevent spoofing or re-routing attacks.
	aad := []byte(senderID + ":" + recipientID)
	encryptedKey, encryptedData, err := crypto.Encrypt(payloadBytes, aad, recipientPubKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt payload: %w", err)
	}

	signature, err := crypto.Sign(encryptedData, privateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to sign encrypted data: %w", err)
	}

	// 4. Create the SecureEnvelope and Send it
	envelope := &transport.SecureEnvelope{
		SenderID:              senderID,
		RecipientID:           recipientID,
		EncryptedSymmetricKey: encryptedKey,
		EncryptedData:         encryptedData,
		Signature:             signature,
	}

	if err := a.RouteClient.Send(ctx, envelope); err != nil {
		return fmt.Errorf("failed to send envelope via routing service: %w", err)
	}

	logger.Info().Msg("Successfully completed intention sharing workflow")
	return nil
}

// buildSharedPayload gathers an intention and all its related data into a portable struct.
func (a *App) buildSharedPayload(ctx context.Context, intentionID uuid.UUID) (*sharing.SharedPayload, error) {
	// For simplicity, we query for a single intention by its ID.
	// A real implementation might use the intention service here.
	querySpec := intentions.QuerySpec{} // Simplified for this example
	allIntentions, err := a.IntentionSvc.GetStore().Query(ctx, querySpec)
	if err != nil {
		return nil, err
	}
	var targetIntention intentions.Intention
	found := false
	for _, intent := range allIntentions {
		if intent.ID == intentionID {
			targetIntention = intent
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("intention with ID %s not found", intentionID)
	}

	payload := &sharing.SharedPayload{
		Intention: targetIntention,
		Locations: make(map[string]locations.Location),
		People:    make(map[string]people.Person),
		Groups:    make(map[string]people.Group),
	}

	// Gather all related locations, people, and groups from the intention's targets.
	for _, target := range targetIntention.Targets {
		switch t := target.(type) {
		case intentions.LocationTarget:
			loc, err := a.LocationSvc.GetLocation(ctx, t.LocationID)
			if err == nil {
				payload.Locations[loc.ID.String()] = loc
			}
		case intentions.ProximityTarget:
			for _, personID := range t.PersonIDs {
				p, err := a.PersonSvc.GetPerson(ctx, personID)
				if err == nil {
					payload.People[p.ID.String()] = p
				}
			}
			for _, groupID := range t.GroupIDs {
				g, err := a.PersonSvc.GetGroup(ctx, groupID)
				if err == nil {
					payload.Groups[g.ID.String()] = g
				}
			}
		}
	}

	return payload, nil
}
