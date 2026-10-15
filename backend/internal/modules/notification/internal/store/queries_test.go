package store

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
)

// TestQueries runs every device query against a migrated database.
func TestQueries(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()
	alice, bob := ids.New(), ids.New()
	for i, id := range []uuid.UUID{alice, bob} {
		if _, err := pool.Exec(ctx, `INSERT INTO identity.users (id, firebase_uid, age_verified_at, last_seen_at, created_at, updated_at)
			VALUES ($1, $2, now(), now(), now(), now())`, id, "fb-"+string(rune('a'+i))); err != nil {
			t.Fatal(err)
		}
	}
	q := sqlc.New(pool)
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	token := "fcm-1"
	upsert := func(user uuid.UUID, at time.Time) {
		t.Helper()
		if err := q.UpsertDevice(ctx, sqlc.UpsertDeviceParams{
			ID: ids.New(), UserID: user, InstallationID: "inst-1", Platform: "android",
			FcmToken: &token, AppVersion: "1.0.0+1", Locale: "tr", Now: at,
		}); err != nil {
			t.Fatal(err)
		}
	}

	upsert(alice, now)
	if n, err := q.RevokeDevice(ctx, sqlc.RevokeDeviceParams{UserID: alice, InstallationID: "inst-1", Now: now}); err != nil || n != 1 {
		t.Fatalf("revoke = %d, %v", n, err)
	}
	if n, _ := q.RevokeDevice(ctx, sqlc.RevokeDeviceParams{UserID: alice, InstallationID: "inst-1", Now: now}); n != 0 {
		t.Fatal("second revoke changed a row")
	}

	// Same installation signs in as bob: the row moves to bob and is active again (D-015).
	upsert(bob, now.Add(time.Minute))
	if list, _ := q.ListActiveDevices(ctx, alice); len(list) != 0 {
		t.Fatalf("alice still has %d devices", len(list))
	}
	list, err := q.ListActiveDevices(ctx, bob)
	if err != nil || len(list) != 1 || list[0].RevokedAt != nil || list[0].FcmToken == nil || *list[0].FcmToken != token {
		t.Fatalf("bob's devices = %+v, %v", list, err)
	}
}
