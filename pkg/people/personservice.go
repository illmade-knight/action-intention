// FILE: people/service.go

package people

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Service provides logic for managing people and groups.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreatePerson adds a new person to the system.
func (s *Service) CreatePerson(ctx context.Context, name string) (Person, error) {
	p := Person{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now(),
	}
	err := s.store.AddPerson(ctx, p)
	return p, err
}

// CreateGroup adds a new group.
func (s *Service) CreateGroup(ctx context.Context, name string) (Group, error) {
	g := Group{
		ID:        uuid.New(),
		Name:      name,
		MemberIDs: []uuid.UUID{},
		CreatedAt: time.Now(),
	}
	err := s.store.AddGroup(ctx, g)
	return g, err
}

func (s *Service) AddMemberToGroup(ctx context.Context, groupID, personID uuid.UUID) error {
	return s.store.AddMemberToGroup(ctx, groupID, personID)
}

func (s *Service) GetPerson(ctx context.Context, id uuid.UUID) (Person, error) {
	return s.store.GetPerson(ctx, id)
}

func (s *Service) GetGroup(ctx context.Context, id uuid.UUID) (Group, error) {
	return s.store.GetGroup(ctx, id)
}

func (s *Service) GetStore() Store {
	return s.store
}
