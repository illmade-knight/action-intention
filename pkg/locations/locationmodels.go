// FILE: locations/models.go

package locations

import (
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LocationType distinguishes between different kinds of locations.
type LocationType string

const (
	LocationTypeUser   LocationType = "USER"   // A private, user-specific location (e.g., "My House").
	LocationTypeShared LocationType = "SHARED" // A public, shared location (e.g., "Fairview Park").
)

// MatchResult indicates the confidence of a match.
type MatchResult string

const (
	MatchNone     MatchResult = "NONE"
	MatchPossible MatchResult = "POSSIBLE"
	MatchExact    MatchResult = "EXACT"
)

// LocationMatcher holds de-normalized data used to find a match.
type LocationMatcher struct {
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Lat      *float64 `json:"lat,omitempty"`
	Lon      *float64 `json:"lon,omitempty"`
}

// Match compares the matcher against a local location to determine match quality.
func (m *LocationMatcher) Match(local Location) MatchResult {
	log.Printf("[Matcher Debug] Comparing INCOMING ('%s') vs LOCAL ('%s')", m.Name, local.Matcher.Name)

	if !strings.EqualFold(m.Name, local.Matcher.Name) {
		log.Println("[Matcher Debug] -> Name MISMATCH. Result: NONE")
		return MatchNone
	}
	log.Println("[Matcher Debug] -> Name MATCH.")

	if m.Lat != nil && m.Lon != nil && local.Matcher.Lat != nil && local.Matcher.Lon != nil {
		distanceKm := haversine(*m.Lat, *m.Lon, *local.Matcher.Lat, *local.Matcher.Lon)
		if distanceKm <= 0.05 {
			return MatchExact
		}
		if distanceKm > 0.5 {
			return MatchNone
		}
		return MatchPossible
	}

	if strings.EqualFold(m.Category, local.Matcher.Category) {
		log.Println("[Matcher Debug] -> Category MATCH. Result: EXACT")
		return MatchExact
	}

	log.Println("[Matcher Debug] -> Category MISMATCH, but name matched. Result: POSSIBLE")
	return MatchPossible
}

// Location represents a physical place in the system.
type Location struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Category  string          `json:"category"`
	GlobalID  *string         `json:"global_id,omitempty"` // For public, shared entities
	Matcher   LocationMatcher `json:"matcher"`             // For matching user-generated entities
	Type      LocationType    `json:"type"`
	UserID    *string         `json:"user_id,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
