// FILE: people/store.go

package people

import (
	"context"

	"github.com/google/uuid"
)

// Store is the interface for storing people and groups.
type Store interface {
	AddPerson(ctx context.Context, p Person) error
	GetPerson(ctx context.Context, id uuid.UUID) (Person, error)
	AddGroup(ctx context.Context, g Group) error
	GetGroup(ctx context.Context, id uuid.UUID) (Group, error)
	AddMemberToGroup(ctx context.Context, groupID, personID uuid.UUID) error
	FindByGlobalID(ctx context.Context, globalID string) (Person, error)
	ListAllForMatching(ctx context.Context) ([]Person, error)
}
