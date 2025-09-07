// FILE: main_distributed.go
// This demo simulates the sharing and reconciliation process between two users.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/action-intention/pkg/people"
	"github.com/illmade-knight/action-intention/pkg/reconciliation"
	"github.com/illmade-knight/action-intention/pkg/sharing"
)

func main() {
	log.Println("--- Starting Distributed Reconciliation Demo ---")
	ctx := context.Background()

	// --- SETUP: Simulate two separate users, Sora and Lucas, with their own local data ---
	// Sora's local data stores
	soraLocationStore := locations.NewInMemoryStore()
	soraPeopleStore := people.NewInMemoryStore()

	// Lucas's local data stores
	lucasLocationStore := locations.NewInMemoryStore()
	lucasPeopleStore := people.NewInMemoryStore()

	// --- SEED DATA for Sora ---
	log.Println("\n--- Populating Sora's local data ---")
	// A shared, public park with a GlobalID
	fairviewGlobalID := "g-loc-fairview-park-dublin"
	soraPark, _ := locations.NewService(soraLocationStore).AddSharedLocation(ctx, "Fairview Park", "Park")
	soraPark.GlobalID = &fairviewGlobalID // Assign the global ID

	// A private, user-specific location with no GlobalID, only a Matcher
	soraCoffee, _ := locations.NewService(soraLocationStore).AddUserLocation(ctx, "Sora", "The Central Perk", "Coffee Shop")

	// A person Sora knows
	soraJim, _ := people.NewService(soraPeopleStore).CreatePerson(ctx, "Jim")
	log.Printf("Sora knows: '%s' (local ID: %s) and '%s' (local ID: %s)", soraPark.Matcher.Name, soraPark.ID, soraCoffee.Matcher.Name, soraCoffee.ID)

	// --- SEED DATA for Lucas ---
	log.Println("\n--- Populating Lucas's local data ---")
	// Lucas also knows about the same public park, with the same GlobalID but a DIFFERENT local ID.
	lucasPark, _ := locations.NewService(lucasLocationStore).AddSharedLocation(ctx, "Fairview Park", "Park")
	lucasPark.GlobalID = &fairviewGlobalID

	// Lucas knows a conceptually similar coffee shop, but calls it something different.
	lucasCoffee, _ := locations.NewService(lucasLocationStore).AddUserLocation(ctx, "Lucas", "The Central Perk", "Coffee Shop")
	log.Printf("Lucas knows: '%s' (local ID: %s) and '%s' (local ID: %s)", lucasPark.Matcher.Name, lucasPark.ID, lucasCoffee.Matcher.Name, lucasCoffee.ID)

	// --- Sora creates and shares an intention ---
	log.Println("\n--- Sora shares an intention: 'Meet Jim at The Central Perk' ---")
	soraIntention := intentions.Intention{
		ID:     uuid.New(),
		User:   "Sora",
		Action: "Meet for coffee",
		Targets: []intentions.Target{
			intentions.LocationTarget{LocationID: soraCoffee.ID},
			intentions.ProximityTarget{PersonIDs: []uuid.UUID{soraJim.ID}},
		},
	}

	// Build the sharable payload
	payload := sharing.SharedPayload{
		Intention: soraIntention,
		Locations: map[string]locations.Location{soraCoffee.ID.String(): soraCoffee},
		People:    map[string]people.Person{soraJim.ID.String(): soraJim},
	}

	// --- Lucas receives the payload and reconciles it ---
	log.Println("\n--- Lucas's system receives the payload and runs the Reconciler ---")

	// The Reconciler is initialized with LUCAS'S local data stores.
	reconciler := reconciliation.NewReconciler(lucasLocationStore, lucasPeopleStore)
	mappingResult, err := reconciler.ProcessPayload(ctx, payload)
	if err != nil {
		log.Fatalf("Reconciliation failed: %v", err)
	}

	// --- Display the results ---
	log.Println("\n--- Reconciliation Result ---")

	// Check the location mapping
	soraCoffeeID := soraCoffee.ID
	if lucasMappedID, ok := mappingResult.LocationMappings[soraCoffeeID]; ok {
		fmt.Printf("✅ SUCCESS: Sora's location '%s' (ID: %s) was matched to Lucas's local location (ID: %s)\n", soraCoffee.Matcher.Name, soraCoffeeID, lucasMappedID)
	} else {
		fmt.Printf("❌ FAILED: Sora's location '%s' was NOT matched.\n", soraCoffee.Matcher.Name)
	}

	// Check the person mapping
	soraJimID := soraJim.ID
	if _, ok := mappingResult.PersonMappings[soraJimID]; ok {
		fmt.Printf("❌ FAILED: Sora's person 'Jim' was matched, but shouldn't have been.\n")
	} else {
		fmt.Printf("✅ SUCCESS: Sora's person 'Jim' (ID: %s) was correctly NOT found in Lucas's data.\n", soraJimID)
	}
}
