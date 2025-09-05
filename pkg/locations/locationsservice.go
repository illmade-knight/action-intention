// FILE: locations/service.go

package locations

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Service provides the business logic for managing locations.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// AddUserLocation creates a new location private to a specific user.
func (s *Service) AddUserLocation(ctx context.Context, userID, name, category string) (Location, error) {
	loc := Location{
		ID:        uuid.New(),
		Name:      name,
		Category:  category,
		Matcher:   LocationMatcher{Name: name, Category: category},
		Type:      LocationTypeUser,
		UserID:    &userID,
		CreatedAt: time.Now(),
	}
	err := s.store.Add(ctx, loc)
	return loc, err
}

// AddSharedLocation creates a new public location available to everyone.
func (s *Service) AddSharedLocation(ctx context.Context, name, category string) (Location, error) {
	loc := Location{
		ID:        uuid.New(),
		Name:      name,
		Category:  category,
		Matcher:   LocationMatcher{Name: name, Category: category},
		Type:      LocationTypeShared,
		UserID:    nil, // No specific owner
		CreatedAt: time.Now(),
	}
	err := s.store.Add(ctx, loc)
	return loc, err
}

// GetLocation fetches a single location by its ID.
func (s *Service) GetLocation(ctx context.Context, id uuid.UUID) (Location, error) {
	return s.store.GetByID(ctx, id)
}
