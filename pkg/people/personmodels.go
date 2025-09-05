// FILE: people/models.go

package people

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// MatchResult mirrors the one in the locations package.
type MatchResult string

const (
	MatchNone     MatchResult = "NONE"
	MatchPossible MatchResult = "POSSIBLE"
	MatchExact    MatchResult = "EXACT"
)

// PersonMatcher holds data for finding a corresponding person on another system.
type PersonMatcher struct {
	Name   string  `json:"name"`
	Handle *string `json:"handle,omitempty"` // e.g., email or phone
}

// Match compares the matcher against a local person.
func (m *PersonMatcher) Match(local Person) MatchResult {
	// Strategy 1: An exact, case-insensitive match on the handle is definitive.
	if m.Handle != nil && local.Matcher.Handle != nil {
		if strings.EqualFold(*m.Handle, *local.Matcher.Handle) {
			return MatchExact
		}
	}

	// Strategy 2: An exact, case-insensitive match on the name is a strong possibility.
	if strings.EqualFold(m.Name, local.Matcher.Name) {
		return MatchPossible
	}

	return MatchNone
}

// Person represents an individual. This is distinct from a system User,
// though they can be linked.

type Person struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	GlobalID  *string       `json:"global_id,omitempty"`
	Matcher   PersonMatcher `json:"matcher"`
	UserID    *string       `json:"user_id,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

// Group represents a collection of people.
type Group struct {
	ID        uuid.UUID
	Name      string
	MemberIDs []uuid.UUID // A list of Person IDs.
	CreatedAt time.Time
}
