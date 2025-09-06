// FILE: pkg/reconciliation/reconciler.go

package reconciliation

import (
	"context"
	"log"

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
	log.Printf("[Reconciler] Found %d local locations for matching.", len(localLocations))
	for senderIDkey, incomingLoc := range payload.Locations {
		senderID, _ := uuid.Parse(senderIDkey)
		log.Printf("[Reconciler] --- Processing incoming location '%s' (ID: %s) ---", incomingLoc.Name, senderID)

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
				log.Printf("[Reconciler] -> Checking against local location: '%s' (ID: %s)", localLoc.Name, localLoc.ID)
				matchLevel := incomingLoc.Matcher.Match(localLoc)
				if matchLevel == locations.MatchExact {
					bestMatchID = &localLoc.ID
					log.Printf("[Reconciler] ----> Found EXACT match!")
					break
				}
				if matchLevel == locations.MatchPossible && bestMatchLevel == locations.MatchNone {
					bestMatchID = &localLoc.ID
					bestMatchLevel = locations.MatchPossible
					log.Printf("[Reconciler] ----> Found POSSIBLE match.")
				}
			}
			if bestMatchID != nil {
				finalMatchID = bestMatchID
			}
		}

		if finalMatchID != nil {
			result.LocationMappings[senderID] = *finalMatchID
			log.Printf("[Reconciler] ===> MAPPED incoming location %s to local %s", senderID, *finalMatchID)
		} else {
			log.Printf("[Reconciler] ===> NO MATCH FOUND for incoming location %s", senderID)
		}
	}

	// --- Reconcile People ---
	localPeople, _ := r.localPersonStore.ListAllForMatching(ctx)
	log.Printf("[Reconciler] Found %d local people for matching.", len(localPeople))
	for senderIDkey, incomingPerson := range payload.People {
		log.Printf("[Reconciler] --- Processing incoming person '%s' (ID: %s) ---", incomingPerson.Name, senderIDkey)
		senderID, _ := uuid.Parse(senderIDkey)

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
				log.Printf("[Reconciler] -> Checking against local person: '%s' (ID: %s)", localPerson.Name, localPerson.ID)
				matchLevel := incomingPerson.Matcher.Match(localPerson)
				if matchLevel == people.MatchExact {
					bestMatchID = &localPerson.ID
					log.Printf("[Reconciler] ----> Found EXACT match!")
					break
				}
				if matchLevel == people.MatchPossible && bestMatchLevel == people.MatchNone {
					bestMatchID = &localPerson.ID
					bestMatchLevel = people.MatchPossible
					log.Printf("[Reconciler] ----> Found POSSIBLE match.")
				}
			}
			if bestMatchID != nil {
				finalMatchID = bestMatchID
			}
		}

		if finalMatchID != nil {
			result.PersonMappings[senderID] = *finalMatchID
			log.Printf("[Reconciler] ===> MAPPED incoming person %s to local %s", senderIDkey, *finalMatchID)
		} else {
			log.Printf("[Reconciler] ===> NO MATCH FOUND for incoming person %s", senderIDkey)
		}
	}

	return result, nil
}
