// FILE: pkg/people/inmem_store.go

package people

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type InMemoryStore struct {
	sync.RWMutex
	people map[uuid.UUID]Person
	groups map[uuid.UUID]Group
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		people: make(map[uuid.UUID]Person),
		groups: make(map[uuid.UUID]Group),
	}
}

// --- Person Methods ---

func (s *InMemoryStore) AddPerson(ctx context.Context, p Person) error {
	s.Lock()
	defer s.Unlock()
	s.people[p.ID] = p
	return nil
}

func (s *InMemoryStore) GetPerson(ctx context.Context, id uuid.UUID) (Person, error) {
	s.RLock()
	defer s.RUnlock()
	p, ok := s.people[id]
	if !ok {
		return Person{}, fmt.Errorf("person %s not found", id)
	}
	return p, nil
}

func (s *InMemoryStore) FindByGlobalID(ctx context.Context, globalID string) (Person, error) {
	s.RLock()
	defer s.RUnlock()
	for _, p := range s.people {
		if p.GlobalID != nil && *p.GlobalID == globalID {
			return p, nil
		}
	}
	return Person{}, fmt.Errorf("person with global ID %s not found", globalID)
}

func (s *InMemoryStore) ListAllForMatching(ctx context.Context) ([]Person, error) {
	s.RLock()
	defer s.RUnlock()
	allPeople := make([]Person, 0, len(s.people))
	for _, p := range s.people {
		allPeople = append(allPeople, p)
	}
	return allPeople, nil
}

// --- Group Methods ---

func (s *InMemoryStore) AddGroup(ctx context.Context, g Group) error {
	s.Lock()
	defer s.Unlock()
	s.groups[g.ID] = g
	return nil
}

func (s *InMemoryStore) GetGroup(ctx context.Context, id uuid.UUID) (Group, error) {
	s.RLock()
	defer s.RUnlock()
	g, ok := s.groups[id]
	if !ok {
		return Group{}, fmt.Errorf("group with ID %s not found", id)
	}
	return g, nil
}

func (s *InMemoryStore) AddMemberToGroup(ctx context.Context, groupID, personID uuid.UUID) error {
	s.Lock()
	defer s.Unlock()
	g, ok := s.groups[groupID]
	if !ok {
		return fmt.Errorf("group with ID %s not found", groupID)
	}
	for _, memberID := range g.MemberIDs {
		if memberID == personID {
			return nil
		}
	}
	g.MemberIDs = append(g.MemberIDs, personID)
	s.groups[groupID] = g
	return nil
}
