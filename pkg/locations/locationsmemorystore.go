// FILE: pkg/locations/inmem_store.go

package locations

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// InMemoryStore is a thread-safe, in-memory implementation of the Store interface.
type InMemoryStore struct {
	sync.RWMutex
	locations map[uuid.UUID]Location
}

// NewInMemoryStore creates a new in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		locations: make(map[uuid.UUID]Location),
	}
}

// Add saves a new location to the store.
func (s *InMemoryStore) Add(ctx context.Context, loc Location) error {
	s.Lock()
	defer s.Unlock()
	s.locations[loc.ID] = loc
	return nil
}

// GetByID retrieves a location by its local UUID.
func (s *InMemoryStore) GetByID(ctx context.Context, id uuid.UUID) (Location, error) {
	s.RLock()
	defer s.RUnlock()
	loc, ok := s.locations[id]
	if !ok {
		return Location{}, fmt.Errorf("location with ID %s not found", id)
	}
	return loc, nil
}

// FindByGlobalID retrieves a location by its public, shared identifier.
func (s *InMemoryStore) FindByGlobalID(ctx context.Context, globalID string) (Location, error) {
	s.RLock()
	defer s.RUnlock()
	for _, loc := range s.locations {
		if loc.GlobalID != nil && *loc.GlobalID == globalID {
			return loc, nil
		}
	}
	return Location{}, fmt.Errorf("location with global ID %s not found", globalID)
}

// ListAllForMatching returns all locations for the purpose of running matcher logic.
func (s *InMemoryStore) ListAllForMatching(ctx context.Context) ([]Location, error) {
	s.RLock()
	defer s.RUnlock()

	allLocations := make([]Location, 0, len(s.locations))
	for _, loc := range s.locations {
		allLocations = append(allLocations, loc)
	}
	return allLocations, nil
}

// (Stubs for other list methods as they are not needed for the reconciler yet)
func (s *InMemoryStore) ListByUserID(ctx context.Context, userID string) ([]Location, error) {
	return nil, nil
}
func (s *InMemoryStore) ListShared(ctx context.Context) ([]Location, error) { return nil, nil }
