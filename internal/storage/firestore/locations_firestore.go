// Package firestore provides persistent storage implementations using Google Cloud Firestore.
package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// locationDocument is the private struct used for Firestore marshalling. This keeps
// the public domain model in `pkg/locations` clean from persistence-specific tags.
type locationDocument struct {
	Name      string                    `firestore:"name"`
	Category  string                    `firestore:"category"`
	GlobalID  *string                   `firestore:"globalId,omitempty"`
	Matcher   locations.LocationMatcher `firestore:"matcher"`
	Type      locations.LocationType    `firestore:"type"`
	UserID    *string                   `firestore:"userId,omitempty"`
	CreatedAt time.Time                 `firestore:"createdAt"`
}

// LocationsStore is a concrete implementation of the locations.Store interface using Firestore.
type LocationsStore struct {
	client     *firestore.Client
	collection *firestore.CollectionRef
}

// NewLocationsStore creates a new Firestore-backed store for locations.
func NewLocationsStore(client *firestore.Client) *LocationsStore {
	return &LocationsStore{
		client:     client,
		collection: client.Collection("locations"),
	}
}

func toLocationDocument(loc locations.Location) locationDocument {
	return locationDocument{
		Name:      loc.Name,
		Category:  loc.Category,
		GlobalID:  loc.GlobalID,
		Matcher:   loc.Matcher,
		Type:      loc.Type,
		UserID:    loc.UserID,
		CreatedAt: loc.CreatedAt,
	}
}

func toLocation(docID uuid.UUID, doc locationDocument) locations.Location {
	return locations.Location{
		ID:        docID,
		Name:      doc.Name,
		Category:  doc.Category,
		GlobalID:  doc.GlobalID,
		Matcher:   doc.Matcher,
		Type:      doc.Type,
		UserID:    doc.UserID,
		CreatedAt: doc.CreatedAt,
	}
}

// Add saves a new location to the store.
func (s *LocationsStore) Add(ctx context.Context, loc locations.Location) error {
	doc := s.collection.Doc(loc.ID.String())
	_, err := doc.Set(ctx, toLocationDocument(loc))
	return err
}

// GetByID retrieves a location by its local UUID.
func (s *LocationsStore) GetByID(ctx context.Context, id uuid.UUID) (locations.Location, error) {
	doc, err := s.collection.Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return locations.Location{}, fmt.Errorf("location with ID %s not found", id)
		}
		return locations.Location{}, err
	}

	var ld locationDocument
	if err := doc.DataTo(&ld); err != nil {
		return locations.Location{}, err
	}
	return toLocation(id, ld), nil
}

// ListByUserID retrieves all locations for a specific user.
func (s *LocationsStore) ListByUserID(ctx context.Context, userID string) ([]locations.Location, error) {
	iter := s.collection.Where("userId", "==", userID).Documents(ctx)
	return processLocationIterator(iter)
}

// ListShared retrieves all public, shared locations.
func (s *LocationsStore) ListShared(ctx context.Context) ([]locations.Location, error) {
	iter := s.collection.Where("type", "==", locations.LocationTypeShared).Documents(ctx)
	return processLocationIterator(iter)
}

// FindByGlobalID retrieves a location by its public, shared identifier.
func (s *LocationsStore) FindByGlobalID(ctx context.Context, globalID string) (locations.Location, error) {
	iter := s.collection.Where("globalId", "==", globalID).Limit(1).Documents(ctx)
	results, err := processLocationIterator(iter)
	if err != nil {
		return locations.Location{}, err
	}
	if len(results) == 0 {
		return locations.Location{}, fmt.Errorf("location with global ID %s not found", globalID)
	}
	return results[0], nil
}

// ListAllForMatching returns all locations for the purpose of running matcher logic.
func (s *LocationsStore) ListAllForMatching(ctx context.Context) ([]locations.Location, error) {
	iter := s.collection.Documents(ctx)
	return processLocationIterator(iter)
}

// processLocationIterator is a helper to drain results from a Firestore iterator.
func processLocationIterator(iter *firestore.DocumentIterator) ([]locations.Location, error) {
	var results []locations.Location
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var ld locationDocument
		if err := doc.DataTo(&ld); err != nil {
			return nil, err
		}
		docID, err := uuid.Parse(doc.Ref.ID)
		if err != nil {
			return nil, err // Should not happen if we control IDs
		}
		results = append(results, toLocation(docID, ld))
	}
	return results, nil
}
