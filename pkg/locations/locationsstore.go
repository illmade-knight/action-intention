// FILE: locations/store.go

package locations

import (
	"context"

	"github.com/google/uuid"
)

// Store is the interface for storing and retrieving locations.
type Store interface {
	Add(ctx context.Context, loc Location) error
	GetByID(ctx context.Context, id uuid.UUID) (Location, error)
	ListByUserID(ctx context.Context, userID string) ([]Location, error)
	ListShared(ctx context.Context) ([]Location, error)
	FindByGlobalID(ctx context.Context, globalID string) (Location, error)
	ListAllForMatching(ctx context.Context) ([]Location, error)
}
