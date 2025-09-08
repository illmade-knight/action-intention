// Package firestore provides persistent storage implementations using Google Cloud Firestore.
package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"google.golang.org/api/iterator"
)

// targetDocument is a private struct for Firestore marshalling that handles the Target interface.
type targetDocument struct {
	Type       string      `firestore:"type"`
	LocationID *uuid.UUID  `firestore:"locationId,omitempty"`
	PersonIDs  []uuid.UUID `firestore:"personIds,omitempty"`
	GroupIDs   []uuid.UUID `firestore:"groupIds,omitempty"`
}

// intentionDocument is the private struct that is actually stored in Firestore.
type intentionDocument struct {
	User         string           `firestore:"user"`
	Participants []string         `firestore:"participants"`
	Action       string           `firestore:"action"`
	Targets      []targetDocument `firestore:"targets"`
	StartTime    time.Time        `firestore:"startTime"`
	EndTime      time.Time        `firestore:"endTime"`
	CreatedAt    time.Time        `firestore:"createdAt"`
}

// IntentionStore is a concrete implementation of the intentions.Store interface using Firestore.
type IntentionStore struct {
	client     *firestore.Client
	collection *firestore.CollectionRef
}

// NewIntentionsStore creates a new Firestore-backed store for intentions.
func NewIntentionsStore(client *firestore.Client) *IntentionStore {
	return &IntentionStore{
		client:     client,
		collection: client.Collection("intentions"),
	}
}

// toTargetDocument converts a Target interface into its serializable document form.
func toTargetDocument(target intentions.Target) (targetDocument, error) {
	switch t := target.(type) {
	case intentions.LocationTarget:
		return targetDocument{Type: "Location", LocationID: &t.LocationID}, nil
	case intentions.ProximityTarget:
		return targetDocument{Type: "Proximity", PersonIDs: t.PersonIDs, GroupIDs: t.GroupIDs}, nil
	default:
		return targetDocument{}, fmt.Errorf("unknown target type: %s", t.Type())
	}
}

// toTarget converts a targetDocument back into a Target interface.
func toTarget(doc targetDocument) (intentions.Target, error) {
	switch doc.Type {
	case "Location":
		if doc.LocationID == nil {
			return nil, fmt.Errorf("location target document has nil LocationID")
		}
		return intentions.LocationTarget{LocationID: *doc.LocationID}, nil
	case "Proximity":
		return intentions.ProximityTarget{PersonIDs: doc.PersonIDs, GroupIDs: doc.GroupIDs}, nil
	default:
		return nil, fmt.Errorf("unknown target type in document: %s", doc.Type)
	}
}

// Add saves a new intention to the store.
func (s *IntentionStore) Add(ctx context.Context, intent intentions.Intention) error {
	targetDocs := make([]targetDocument, len(intent.Targets))
	for i, target := range intent.Targets {
		doc, err := toTargetDocument(target)
		if err != nil {
			return err
		}
		targetDocs[i] = doc
	}

	doc := intentionDocument{
		User:         intent.User,
		Participants: intent.Participants,
		Action:       intent.Action,
		Targets:      targetDocs,
		StartTime:    intent.StartTime,
		EndTime:      intent.EndTime,
		CreatedAt:    intent.CreatedAt,
	}

	_, err := s.collection.Doc(intent.ID.String()).Set(ctx, doc)
	return err
}

// Query retrieves intentions based on the provided specification.
func (s *IntentionStore) Query(ctx context.Context, spec intentions.QuerySpec) ([]intentions.Intention, error) {
	q := s.collection.Query
	if spec.User != nil {
		q = q.Where("user", "==", *spec.User)
	}
	if spec.ActiveAt != nil {
		q = q.Where("startTime", "<=", *spec.ActiveAt).Where("endTime", ">=", *spec.ActiveAt)
	}

	iter := q.Documents(ctx)
	var results []intentions.Intention
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var idoc intentionDocument
		if err := doc.DataTo(&idoc); err != nil {
			return nil, err
		}

		targets := make([]intentions.Target, len(idoc.Targets))
		for i, tDoc := range idoc.Targets {
			target, err := toTarget(tDoc)
			if err != nil {
				return nil, err
			}
			targets[i] = target
		}

		docID, err := uuid.Parse(doc.Ref.ID)
		if err != nil {
			return nil, err
		}

		results = append(results, intentions.Intention{
			ID:           docID,
			User:         idoc.User,
			Participants: idoc.Participants,
			Action:       idoc.Action,
			Targets:      targets,
			StartTime:    idoc.StartTime,
			EndTime:      idoc.EndTime,
			CreatedAt:    idoc.CreatedAt,
		})
	}
	return results, nil
}
