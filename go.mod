module github.com/illmade-knight/action-intention

go 1.24

require (
	github.com/google/uuid v1.6.0
	github.com/illmade-knight/go-key-service v0.0.0
	github.com/illmade-knight/go-secure-messaging v0.0.0
	github.com/illmade-knight/routing-service v0.0.0
)

replace github.com/illmade-knight/go-secure-messaging => ../go-secure-messaging

// Add the replace block to point to your local directories.
// The paths are relative to this go.mod file.
replace github.com/illmade-knight/go-key-service => ../go-key-service

replace github.com/illmade-knight/routing-service => ../routing-service
