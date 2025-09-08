//go:build integration

package firestore_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	fst "github.com/illmade-knight/action-intention/internal/storage/firestore"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"github.com/illmade-knight/go-test/emulators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupIntentionsTest(t *testing.T) (context.Context, *firestore.Client, *fst.IntentionStore) {
	t.Helper()
	ctx := context.Background()
	fsConn := emulators.SetupFirestoreEmulator(t, ctx, emulators.GetDefaultFirestoreConfig("test-project"))
	fsClient, err := firestore.NewClient(ctx, "test-project", fsConn.ClientOptions...)
	require.NoError(t, err)

	store := fst.NewIntentionsStore(fsClient)
	require.NotNil(t, store)

	t.Cleanup(func() {
		fsClient.Close()
	})
	return ctx, fsClient, store
}

func TestIntentionsStore(t *testing.T) {
	ctx, _, store := setupIntentionsTest(t)
	now := time.Now()

	// Arrange: Create test data
	locTarget := intentions.LocationTarget{LocationID: uuid.New()}
	proxTarget := intentions.ProximityTarget{PersonIDs: []uuid.UUID{uuid.New()}}

	intent1 := intentions.Intention{
		ID:        uuid.New(),
		User:      "user-alice",
		Action:    "Work",
		Targets:   []intentions.Target{locTarget},
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(1 * time.Hour), // Active now
	}
	intent2 := intentions.Intention{
		ID:        uuid.New(),
		User:      "user-alice",
		Action:    "Coffee",
		Targets:   []intentions.Target{proxTarget},
		StartTime: now.Add(2 * time.Hour),
		EndTime:   now.Add(3 * time.Hour), // In the future
	}
	intent3 := intentions.Intention{
		ID:        uuid.New(),
		User:      "user-bob",
		Action:    "Lunch",
		Targets:   []intentions.Target{locTarget},
		StartTime: now.Add(-2 * time.Hour),
		EndTime:   now.Add(-1 * time.Hour), // In the past
	}

	// Act: Add all intentions
	require.NoError(t, store.Add(ctx, intent1))
	require.NoError(t, store.Add(ctx, intent2))
	require.NoError(t, store.Add(ctx, intent3))

	// Assert: Test Query logic
	t.Run("Query by user", func(t *testing.T) {
		user := "user-alice"
		spec := intentions.QuerySpec{User: &user}
		results, err := store.Query(ctx, spec)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("Query by active time", func(t *testing.T) {
		spec := intentions.QuerySpec{ActiveAt: &now}
		results, err := store.Query(ctx, spec)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, intent1.ID, results[0].ID)
	})

	t.Run("Query by user and active time", func(t *testing.T) {
		user := "user-alice"
		spec := intentions.QuerySpec{User: &user, ActiveAt: &now}
		results, err := store.Query(ctx, spec)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, intent1.ID, results[0].ID)
	})

	t.Run("Query with no matches", func(t *testing.T) {
		user := "user-charlie"
		spec := intentions.QuerySpec{User: &user}
		results, err := store.Query(ctx, spec)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}
