package store

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
)

func ptr[T any](v T) *T { return &v }

// TestQueries runs every identity query against a migrated database.
func TestQueries(t *testing.T) {
	q := sqlc.New(testdb.New(t))
	ctx := t.Context()
	t0 := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	id := ids.New()

	if _, err := q.GetUserByFirebaseUID(ctx, "fb-1"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("missing user: %v, want ErrNoRows", err)
	}
	if n, err := q.InsertUser(ctx, sqlc.InsertUserParams{ID: id, FirebaseUid: "fb-1", AccountTier: "guest", Now: t0}); err != nil || n != 1 {
		t.Fatalf("InsertUser = %d, %v", n, err)
	}
	if n, err := q.InsertUser(ctx, sqlc.InsertUserParams{ID: ids.New(), FirebaseUid: "fb-1", AccountTier: "guest", Now: t0}); err != nil || n != 0 {
		t.Fatalf("second InsertUser for the same firebase_uid = %d, %v; want 0 rows", n, err)
	}
	u, err := q.GetUserByFirebaseUID(ctx, "fb-1")
	if err != nil || u.ID != id || u.AccountTier != "guest" || u.Status != "active" || !u.AgeVerifiedAt.Equal(t0) || u.UpgradedAt != nil {
		t.Fatalf("user = %+v, %v", u, err)
	}

	// Guest → unverified → full: upgraded_at set once, kept afterwards.
	t1, t2 := t0.Add(time.Hour), t0.Add(2*time.Hour)
	for _, step := range []struct {
		tier string
		at   time.Time
	}{{"unverified", t0.Add(time.Minute)}, {"full", t1}, {"full", t2}} {
		if err := q.UpdateUserTier(ctx, sqlc.UpdateUserTierParams{ID: id, AccountTier: step.tier, Now: step.at}); err != nil {
			t.Fatal(err)
		}
	}
	u, _ = q.GetUserByID(ctx, id)
	if u.AccountTier != "full" || u.UpgradedAt == nil || !u.UpgradedAt.Equal(t1) || !u.UpdatedAt.Equal(t2) {
		t.Fatalf("after upgrade: tier %s upgraded_at %v updated_at %v", u.AccountTier, u.UpgradedAt, u.UpdatedAt)
	}

	// Identities: insert, then the same provider again updates in place.
	for _, p := range []sqlc.UpsertAuthIdentityParams{
		{UserID: id, Provider: "anonymous", Now: t0},
		{UserID: id, Provider: "password", Email: ptr("Ali@Example.com"), Now: t0},
		{UserID: id, Provider: "password", Email: ptr("Ali@Example.com"), EmailVerified: true, Now: t1},
	} {
		if err := q.UpsertAuthIdentity(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	list, err := q.ListAuthIdentities(ctx, id)
	if err != nil || len(list) != 2 || list[0].Provider != "anonymous" || list[1].Provider != "password" ||
		!list[1].EmailVerified || !list[1].LinkedAt.Equal(t0) || !list[1].LastSignInAt.Equal(t1) {
		t.Fatalf("identities = %+v, %v", list, err)
	}

	// The mirror drops providers no longer linked in Firebase.
	if err := q.DeleteAuthIdentitiesExcept(ctx, sqlc.DeleteAuthIdentitiesExceptParams{UserID: id, Providers: []string{"password"}}); err != nil {
		t.Fatal(err)
	}
	if list, _ := q.ListAuthIdentities(ctx, id); len(list) != 1 || list[0].Provider != "password" {
		t.Fatalf("after DeleteAuthIdentitiesExcept: %+v", list)
	}

	// Only guests can be marked abandoned.
	if n, err := q.MarkGuestAbandoned(ctx, sqlc.MarkGuestAbandonedParams{ID: id, Now: t2}); err != nil || n != 0 {
		t.Fatalf("MarkGuestAbandoned on a full account = %d, %v", n, err)
	}
	if u, _ = q.GetUserByID(ctx, id); u.AbandonedAt != nil {
		t.Fatal("a full account was marked abandoned")
	}

	if err := q.SetDeletionRequested(ctx, sqlc.SetDeletionRequestedParams{ID: id, DeletionRequestedAt: &t2, Now: t2}); err != nil {
		t.Fatal(err)
	}
	if u, _ = q.GetUserByID(ctx, id); u.DeletionRequestedAt == nil {
		t.Fatal("deletion request not stored")
	}
	_ = q.SetDeletionRequested(ctx, sqlc.SetDeletionRequestedParams{ID: id, Now: t2})
	if u, _ = q.GetUserByID(ctx, id); u.DeletionRequestedAt != nil {
		t.Fatal("deletion request not cancelled")
	}

	if err := q.TouchUser(ctx, sqlc.TouchUserParams{ID: id, Now: t2}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.GetUserByFirebaseUIDForUpdate(ctx, "fb-1"); err != nil {
		t.Fatal(err)
	}
	if roles, err := q.ListUserRoles(ctx, id); err != nil || len(roles) != 0 {
		t.Fatalf("roles = %v, %v", roles, err)
	}
}
