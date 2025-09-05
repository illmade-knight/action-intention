// FILE: intentions/models.go

package intentions

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Target represents the object or destination of an intention.
// It's an interface to allow for different kinds of targets like
// physical locations, people, or even online activities.
type Target interface {
	// Type returns a string identifier for the kind of target (e.g., "Location").
	Type() string
	// Description provides a human-readable summary of the target.
	Description() string
}

// --- Concrete Target Implementations ---

// LocationTarget represents a physical place by referencing its unique ID.
// The full location details are managed by the locations package.
type LocationTarget struct {
	LocationID uuid.UUID
}

func (l LocationTarget) Type() string {
	return "Location"
}

// Description now only returns the ID. The service layer will be responsible
// for "hydrating" this with the full location name for display.
func (l LocationTarget) Description() string {
	return fmt.Sprintf("ID: %s", l.LocationID)
}

// ProximityTarget now represents being with specific people or groups by ID.
type ProximityTarget struct {
	PersonIDs []uuid.UUID
	GroupIDs  []uuid.UUID
}

func (p ProximityTarget) Type() string {
	return "Proximity"
}

func (p ProximityTarget) Description() string {
	return fmt.Sprintf("People: %d, Groups: %d", len(p.PersonIDs), len(p.GroupIDs))
}

// --- Main Intention Struct ---

// Intention holds the complete details of a user's plan.
type Intention struct {
	ID           uuid.UUID
	User         string
	Participants []string
	Action       string
	Targets      []Target // Changed from "Target Target" to "Targets []Target"
	StartTime    time.Time
	EndTime      time.Time
	CreatedAt    time.Time
}
