// Package firestore provides persistent storage implementations using Google Cloud Firestore.
package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/people"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// personDocument is the private struct for Firestore marshalling.
type personDocument struct {
	Name      string               `firestore:"name"`
	GlobalID  *string              `firestore:"globalId,omitempty"`
	Matcher   people.PersonMatcher `firestore:"matcher"`
	UserID    *string              `firestore:"userId,omitempty"`
	CreatedAt time.Time            `firestore:"createdAt"`
}

// groupDocument is the private struct for Firestore marshalling.
type groupDocument struct {
	Name      string      `firestore:"name"`
	MemberIDs []uuid.UUID `firestore:"memberIds"`
	CreatedAt time.Time   `firestore:"createdAt"`
}

// LocationsStore is a concrete implementation of the people.Store interface using Firestore.
type PeopleStore struct {
	client           *firestore.Client
	peopleCollection *firestore.CollectionRef
	groupsCollection *firestore.CollectionRef
}

// NewPeopleStore creates a new Firestore-backed store for people and groups.
func NewPeopleStore(client *firestore.Client) *PeopleStore {
	return &PeopleStore{
		client:           client,
		peopleCollection: client.Collection("people"),
		groupsCollection: client.Collection("groups"),
	}
}

// --- Person Methods ---

func (s *PeopleStore) AddPerson(ctx context.Context, p people.Person) error {
	doc := s.peopleCollection.Doc(p.ID.String())
	_, err := doc.Set(ctx, personDocument{
		Name:      p.Name,
		GlobalID:  p.GlobalID,
		Matcher:   p.Matcher,
		UserID:    p.UserID,
		CreatedAt: p.CreatedAt,
	})
	return err
}

func (s *PeopleStore) GetPerson(ctx context.Context, id uuid.UUID) (people.Person, error) {
	doc, err := s.peopleCollection.Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return people.Person{}, fmt.Errorf("person with ID %s not found", id)
		}
		return people.Person{}, err
	}
	var pd personDocument
	if err := doc.DataTo(&pd); err != nil {
		return people.Person{}, err
	}
	return people.Person{
		ID:        id,
		Name:      pd.Name,
		GlobalID:  pd.GlobalID,
		Matcher:   pd.Matcher,
		UserID:    pd.UserID,
		CreatedAt: pd.CreatedAt,
	}, nil
}

func (s *PeopleStore) FindByGlobalID(ctx context.Context, globalID string) (people.Person, error) {
	iter := s.peopleCollection.Where("globalId", "==", globalID).Limit(1).Documents(ctx)
	results, err := processPersonIterator(iter)
	if err != nil {
		return people.Person{}, err
	}
	if len(results) == 0 {
		return people.Person{}, fmt.Errorf("person with global ID %s not found", globalID)
	}
	return results[0], nil
}

func (s *PeopleStore) ListAllForMatching(ctx context.Context) ([]people.Person, error) {
	iter := s.peopleCollection.Documents(ctx)
	return processPersonIterator(iter)
}

// --- Group Methods ---

func (s *PeopleStore) AddGroup(ctx context.Context, g people.Group) error {
	doc := s.groupsCollection.Doc(g.ID.String())
	_, err := doc.Set(ctx, groupDocument{
		Name:      g.Name,
		MemberIDs: g.MemberIDs,
		CreatedAt: g.CreatedAt,
	})
	return err
}

func (s *PeopleStore) GetGroup(ctx context.Context, id uuid.UUID) (people.Group, error) {
	doc, err := s.groupsCollection.Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return people.Group{}, fmt.Errorf("group with ID %s not found", id)
		}
		return people.Group{}, err
	}
	var gd groupDocument
	if err := doc.DataTo(&gd); err != nil {
		return people.Group{}, err
	}
	return people.Group{
		ID:        id,
		Name:      gd.Name,
		MemberIDs: gd.MemberIDs,
		CreatedAt: gd.CreatedAt,
	}, nil
}

func (s *PeopleStore) AddMemberToGroup(ctx context.Context, groupID, personID uuid.UUID) error {
	groupRef := s.groupsCollection.Doc(groupID.String())
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// First, verify the person exists to maintain data integrity.
		_, err := tx.Get(s.peopleCollection.Doc(personID.String()))
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return fmt.Errorf("person with ID %s does not exist", personID)
			}
			return err
		}
		// Now, update the group.
		return tx.Update(groupRef, []firestore.Update{
			{Path: "memberIds", Value: firestore.ArrayUnion(personID.String())},
		})
	})
}

// --- Helper Functions ---
func processPersonIterator(iter *firestore.DocumentIterator) ([]people.Person, error) {
	var results []people.Person
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var pd personDocument
		if err := doc.DataTo(&pd); err != nil {
			return nil, err
		}
		docID, err := uuid.Parse(doc.Ref.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, people.Person{
			ID:        docID,
			Name:      pd.Name,
			GlobalID:  pd.GlobalID,
			Matcher:   pd.Matcher,
			UserID:    pd.UserID,
			CreatedAt: pd.CreatedAt,
		})
	}
	return results, nil
}
