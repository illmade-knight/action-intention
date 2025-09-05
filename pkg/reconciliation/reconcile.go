// FILE: pkg/reconciliation/reconciler.go

package reconciliation

import (
	"context"

	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/action-intention/pkg/people"
	"github.com/illmade-knight/action-intention/pkg/sharing"
)

type MappingResult struct {
	LocationMappings map[uuid.UUID]uuid.UUID
	PersonMappings   map[uuid.UUID]uuid.UUID
}

type Reconciler struct {
	localLocationStore locations.Store
	localPersonStore   people.Store
}

func NewReconciler(locStore locations.Store, personStore people.Store) *Reconciler {
	return &Reconciler{
		localLocationStore: locStore,
		localPersonStore:   personStore,
	}
}

func (r *Reconciler) ProcessPayload(ctx context.Context, payload sharing.SharedPayload) (MappingResult, error) {
	result := MappingResult{
		LocationMappings: make(map[uuid.UUID]uuid.UUID),
		PersonMappings:   make(map[uuid.UUID]uuid.UUID),
	}

	// --- Reconcile Locations ---
	localLocations, _ := r.localLocationStore.ListAllForMatching(ctx)
	for senderID, incomingLoc := range payload.Locations {
		var finalMatchID *uuid.UUID

		if incomingLoc.GlobalID != nil {
			if loc, err := r.localLocationStore.FindByGlobalID(ctx, *incomingLoc.GlobalID); err == nil {
				finalMatchID = &loc.ID
			}
		}

		if finalMatchID == nil {
			var bestMatchID *uuid.UUID
			bestMatchLevel := locations.MatchNone
			for _, localLoc := range localLocations {
				matchLevel := incomingLoc.Matcher.Match(localLoc)
				if matchLevel == locations.MatchExact {
					bestMatchID = &localLoc.ID
					break
				}
				if matchLevel == locations.MatchPossible && bestMatchLevel == locations.MatchNone {
					bestMatchID = &localLoc.ID
					bestMatchLevel = locations.MatchPossible
				}
			}
			if bestMatchID != nil {
				finalMatchID = bestMatchID
			}
		}

		if finalMatchID != nil {
			result.LocationMappings[senderID] = *finalMatchID
		}
	}

	// --- Reconcile People ---
	localPeople, _ := r.localPersonStore.ListAllForMatching(ctx)
	for senderID, incomingPerson := range payload.People {
		var finalMatchID *uuid.UUID

		if incomingPerson.GlobalID != nil {
			if p, err := r.localPersonStore.FindByGlobalID(ctx, *incomingPerson.GlobalID); err == nil {
				finalMatchID = &p.ID
			}
		}

		if finalMatchID == nil {
			var bestMatchID *uuid.UUID
			bestMatchLevel := people.MatchNone
			for _, localPerson := range localPeople {
				matchLevel := incomingPerson.Matcher.Match(localPerson)
				if matchLevel == people.MatchExact {
					bestMatchID = &localPerson.ID
					break
				}
				if matchLevel == people.MatchPossible && bestMatchLevel == people.MatchNone {
					bestMatchID = &localPerson.ID
					bestMatchLevel = people.MatchPossible
				}
			}
			if bestMatchID != nil {
				finalMatchID = bestMatchID
			}
		}

		if finalMatchID != nil {
			result.PersonMappings[senderID] = *finalMatchID
		}
	}

	return result, nil
}
