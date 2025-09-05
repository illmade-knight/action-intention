// FILE: intentions/store.go

package intentions

import (
	"context"
	"time"
)

// QuerySpec defines the parameters for a query.
// Using pointers allows us to distinguish between a filter not being set
// and a filter having an empty value.
type QuerySpec struct {
	User     *string
	ActiveAt *time.Time // Find intentions that are active at this specific time.
}

// Store is the interface for storing and retrieving intentions.
// This decouples the service from the database implementation.
type Store interface {
	// Add saves a new intention to the store.
	Add(ctx context.Context, intent Intention) error
	// Query retrieves intentions based on the provided specification.
	Query(ctx context.Context, spec QuerySpec) ([]Intention, error)
}
