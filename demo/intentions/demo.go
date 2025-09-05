// FILE: main.go
// This demo shows the basic, single-user functionality.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/action-intention/pkg/people"
)

func main() {
	log.Println("--- Starting Single-User Demo ---")

	// 1. Initialize all services and their in-memory stores
	locationStore := locations.NewInMemoryStore()
	locationService := locations.NewService(locationStore)
	peopleStore := people.NewInMemoryStore()
	peopleService := people.NewService(peopleStore)
	intentionStore := intentions.NewInMemoryStore()
	intentionService := intentions.NewService(intentionStore)

	ctx := context.Background()
	user := "Sora"

	// 2. Seed Data
	log.Println("\n--- Seeding Data ---")
	pizzaPlace, _ := locationService.AddUserLocation(ctx, user, "4Star Pizza", "Restaurant")
	park, _ := locationService.AddSharedLocation(ctx, "Fairview Park", "Football Pitch")
	jim, _ := peopleService.CreatePerson(ctx, "Jim")
	team, _ := peopleService.CreateGroup(ctx, "The Football Team")
	_ = peopleService.AddMemberToGroup(ctx, team.ID, jim.ID)
	log.Printf("✅ Seeded locations, people, and groups.")

	// 3. Add Intentions (Updated to use a slice for Targets)
	log.Println("\n--- Adding Intentions ---")

	// Intention 1: Active now
	pizzaTargets := []intentions.Target{
		intentions.LocationTarget{LocationID: pizzaPlace.ID},
	}
	pizzaStart := time.Now()
	pizzaEnd := pizzaStart.Add(30 * time.Minute)
	_, _ = intentionService.AddIntention(ctx, user, "get a pizza", pizzaTargets, pizzaStart, pizzaEnd)
	log.Println("✅ Added 'get a pizza' intention.")

	// Intention 2: In the future, with multiple targets
	footballTargets := []intentions.Target{
		intentions.LocationTarget{LocationID: park.ID},
		intentions.ProximityTarget{GroupIDs: []uuid.UUID{team.ID}},
	}
	now := time.Now()
	footballStart := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
	footballEnd := time.Date(now.Year(), now.Month(), now.Day(), 17, 30, 0, 0, now.Location())
	_, _ = intentionService.AddIntention(ctx, user, "play football", footballTargets, footballStart, footballEnd)
	log.Println("✅ Added 'play football' intention.")

	// 4. Query for active intentions and display details
	log.Println("\n--- Querying for active intentions NOW ---")
	activeIntentions, _ := intentionService.GetActiveIntentionsForUser(ctx, user)

	if len(activeIntentions) == 0 {
		log.Println("🤔 No active intentions found for", user)
	} else {
		log.Println("⭐ Active intentions for", user, "⭐")
		for _, intent := range activeIntentions {
			fmt.Printf("\n -> Action: %s\n", intent.Action)
			// Loop through the slice of targets to display them
			for _, target := range intent.Targets {
				switch t := target.(type) {
				case intentions.LocationTarget:
					loc, _ := locationService.GetLocation(ctx, t.LocationID)
					fmt.Printf("    📍 Where: %s\n", loc.Matcher.Name)
				case intentions.ProximityTarget:
					// Hydrating proximity targets would be similar
					fmt.Printf("    👥 Who: with a group\n")
				}
			}
		}
	}
}
