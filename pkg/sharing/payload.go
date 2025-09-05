// FILE: pkg/sharing/payload.go

package sharing

import (
	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/action-intention/pkg/people"
)

// SharedPayload is a self-contained, portable representation of an intention
// and all its related data (the "sub-graph").
type SharedPayload struct {
	Intention intentions.Intention             `json:"intention"`
	Locations map[uuid.UUID]locations.Location `json:"locations"`
	People    map[uuid.UUID]people.Person      `json:"people"`
	Groups    map[uuid.UUID]people.Group       `json:"groups"`
}
