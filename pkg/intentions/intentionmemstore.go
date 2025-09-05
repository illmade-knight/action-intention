// FILE: intentions/inmem_store.go

package intentions

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// InMemoryStore is a thread-safe, in-memory implementation of the Store interface.
type InMemoryStore struct {
	sync.RWMutex
	intentions map[uuid.UUID]Intention
}

// NewInMemoryStore creates a new in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		intentions: make(map[uuid.UUID]Intention),
	}
}

// Add saves a new intention to the store.
func (s *InMemoryStore) Add(ctx context.Context, intent Intention) error {
	s.Lock()
	defer s.Unlock()

	// In a real DB, this might check for duplicate IDs, but UUIDs make that rare.
	s.intentions[intent.ID] = intent
	return nil
}

// Query retrieves intentions based on the provided specification.
func (s *InMemoryStore) Query(ctx context.Context, spec QuerySpec) ([]Intention, error) {
	s.RLock()
	defer s.RUnlock()

	var results []Intention

	for _, intent := range s.intentions {
		// If a user filter is specified, and it doesn't match, skip.
		if spec.User != nil && intent.User != *spec.User {
			continue
		}

		// If an active-at filter is specified, and the intention is not active at that time, skip.
		if spec.ActiveAt != nil {
			t := *spec.ActiveAt
			if t.Before(intent.StartTime) || t.After(intent.EndTime) {
				continue
			}
		}

		results = append(results, intent)
	}

	return results, nil
}
