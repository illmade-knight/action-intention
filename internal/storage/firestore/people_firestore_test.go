//go:build integration

package firestore_test

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	fst "github.com/illmade-knight/action-intention/internal/storage/firestore"
	"github.com/illmade-knight/action-intention/pkg/people"
	"github.com/illmade-knight/go-test/emulators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPeopleTest(t *testing.T) (context.Context, *firestore.Client, *fst.PeopleStore) {
	t.Helper()
	ctx := context.Background()
	fsConn := emulators.SetupFirestoreEmulator(t, ctx, emulators.GetDefaultFirestoreConfig("test-project"))
	fsClient, err := firestore.NewClient(ctx, "test-project", fsConn.ClientOptions...)
	require.NoError(t, err)

	store := fst.NewPeopleStore(fsClient)
	require.NotNil(t, store)

	t.Cleanup(func() {
		fsClient.Close()
	})
	return ctx, fsClient, store
}

func TestPeopleStore(t *testing.T) {
	ctx, _, store := setupPeopleTest(t)
	globalID := "person-abc"

	// Arrange: Create test data
	person1 := people.Person{
		ID:       uuid.New(),
		Name:     "Alice",
		GlobalID: &globalID,
		Matcher:  people.PersonMatcher{Name: "Alice"},
	}
	person2 := people.Person{
		ID:      uuid.New(),
		Name:    "Bob",
		Matcher: people.PersonMatcher{Name: "Bob"},
	}
	group1 := people.Group{
		ID:        uuid.New(),
		Name:      "Developers",
		MemberIDs: []uuid.UUID{},
	}

	// Act & Assert: Add
	err := store.AddPerson(ctx, person1)
	require.NoError(t, err)
	err = store.AddPerson(ctx, person2)
	require.NoError(t, err)
	err = store.AddGroup(ctx, group1)
	require.NoError(t, err)

	t.Run("GetPerson", func(t *testing.T) {
		retrieved, err := store.GetPerson(ctx, person1.ID)
		require.NoError(t, err)
		assert.Equal(t, person1.Name, retrieved.Name)
	})

	t.Run("GetGroup", func(t *testing.T) {
		retrieved, err := store.GetGroup(ctx, group1.ID)
		require.NoError(t, err)
		assert.Equal(t, group1.Name, retrieved.Name)
	})

	t.Run("FindByGlobalID", func(t *testing.T) {
		retrieved, err := store.FindByGlobalID(ctx, globalID)
		require.NoError(t, err)
		assert.Equal(t, person1.ID, retrieved.ID)
	})

	t.Run("ListAllForMatching", func(t *testing.T) {
		all, err := store.ListAllForMatching(ctx)
		require.NoError(t, err)
		assert.Len(t, all, 2)
	})

	t.Run("AddMemberToGroup", func(t *testing.T) {
		// Act
		err := store.AddMemberToGroup(ctx, group1.ID, person1.ID)
		require.NoError(t, err)

		// Assert
		updatedGroup, err := store.GetGroup(ctx, group1.ID)
		require.NoError(t, err)
		require.Len(t, updatedGroup.MemberIDs, 1)
		assert.Equal(t, person1.ID, updatedGroup.MemberIDs[0])

		// Act: Add again (should be idempotent)
		err = store.AddMemberToGroup(ctx, group1.ID, person1.ID)
		require.NoError(t, err)
		idempotentGroup, err := store.GetGroup(ctx, group1.ID)
		require.NoError(t, err)
		assert.Len(t, idempotentGroup.MemberIDs, 1)
	})
}
