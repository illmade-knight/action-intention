//go:build integration

package firestore_test

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	fs "github.com/illmade-knight/action-intention/internal/storage/firestore"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/go-test/emulators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLocationsTest(t *testing.T) (context.Context, *firestore.Client, *fs.PeopleStore) {
	t.Helper()
	ctx := context.Background()
	fsConn := emulators.SetupFirestoreEmulator(t, ctx, emulators.GetDefaultFirestoreConfig("test-project"))
	fsClient, err := firestore.NewClient(ctx, "test-project", fsConn.ClientOptions...)
	require.NoError(t, err)

	store := fs.NewLocationsStore(fsClient)
	require.NotNil(t, store)

	t.Cleanup(func() {
		fsClient.Close()
	})
	return ctx, fsClient, store
}

func TestLocationsStore(t *testing.T) {
	ctx, _, store := setupLocationsTest(t)
	userID := "user-123"

	// Arrange: Create some test locations
	userLoc := locations.Location{
		ID:       uuid.New(),
		Name:     "Home",
		Type:     locations.LocationTypeUser,
		UserID:   &userID,
		Matcher:  locations.LocationMatcher{Name: "Home"},
		Category: "Residence",
	}
	sharedLocID := "place-abc"
	sharedLoc := locations.Location{
		ID:       uuid.New(),
		Name:     "Park",
		Type:     locations.LocationTypeShared,
		GlobalID: &sharedLocID,
		Matcher:  locations.LocationMatcher{Name: "Park"},
		Category: "Recreation",
	}

	// Act & Assert: Add
	err := store.Add(ctx, userLoc)
	require.NoError(t, err)
	err = store.Add(ctx, sharedLoc)
	require.NoError(t, err)

	// Act & Assert: GetByID
	t.Run("GetByID", func(t *testing.T) {
		retrieved, err := store.GetByID(ctx, userLoc.ID)
		require.NoError(t, err)
		assert.Equal(t, userLoc.Name, retrieved.Name)
	})

	// Act & Assert: ListByUserID
	t.Run("ListByUserID", func(t *testing.T) {
		userLocations, err := store.ListByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, userLocations, 1)
		assert.Equal(t, userLoc.ID, userLocations[0].ID)
	})

	// Act & Assert: ListShared
	t.Run("ListShared", func(t *testing.T) {
		sharedLocations, err := store.ListShared(ctx)
		require.NoError(t, err)
		require.Len(t, sharedLocations, 1)
		assert.Equal(t, sharedLoc.ID, sharedLocations[0].ID)
	})

	// Act & Assert: FindByGlobalID
	t.Run("FindByGlobalID", func(t *testing.T) {
		foundShared, err := store.FindByGlobalID(ctx, sharedLocID)
		require.NoError(t, err)
		assert.Equal(t, sharedLoc.ID, foundShared.ID)
	})

	// Act & Assert: ListAllForMatching
	t.Run("ListAllForMatching", func(t *testing.T) {
		all, err := store.ListAllForMatching(ctx)
		require.NoError(t, err)
		assert.Len(t, all, 2)
	})
}
