// FILE: intentions/service.go

package intentions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides the business logic for managing intentions.
// It orchestrates the storage and retrieval of intention data.
type Service struct {
	store Store
}

// NewService is the constructor for our intentions Service.
// It takes a Store, allowing us to easily switch between in-memory,
// database, or mock stores.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// AddIntention creates a new intention, validates it, and saves it to the store.
// MODIFIED: The 'target' parameter is now a slice 'targets []Target'.
func (s *Service) AddIntention(ctx context.Context, user, action string, targets []Target, start, end time.Time) (Intention, error) {
	// --- Validation ---
	if user == "" || action == "" {
		return Intention{}, fmt.Errorf("user and action cannot be empty")
	}
	if end.Before(start) {
		return Intention{}, fmt.Errorf("end time cannot be before start time")
	}
	if len(targets) == 0 { // MODIFIED: Validation logic updated
		return Intention{}, fmt.Errorf("at least one target is required")
	}

	intent := Intention{
		ID:        uuid.New(),
		User:      user,
		Action:    action,
		Targets:   targets, // MODIFIED: Assignment updated
		StartTime: start,
		EndTime:   end,
		CreatedAt: time.Now(),
	}

	if err := s.store.Add(ctx, intent); err != nil {
		return Intention{}, fmt.Errorf("failed to save intention: %w", err)
	}
	return intent, nil
}

// GetActiveIntentionsForUser is a convenient method to find what a user is currently doing.
func (s *Service) GetActiveIntentionsForUser(ctx context.Context, user string) ([]Intention, error) {
	now := time.Now()
	spec := QuerySpec{
		User:     &user,
		ActiveAt: &now,
	}

	return s.store.Query(ctx, spec)
}
