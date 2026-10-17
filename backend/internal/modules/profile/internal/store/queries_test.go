package store

import (
	"testing"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
)

// TestQueries runs every profile query against a migrated database.
func TestQueries(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()
	id := ids.New()
	// Test fixture: the owning user row (identity's table, written directly here).
	if _, err := pool.Exec(ctx, `INSERT INTO identity.users (id, firebase_uid, age_verified_at, last_seen_at, created_at, updated_at)
		VALUES ($1, 'fb-1', now(), now(), now(), now())`, id); err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	tr := "TR"

	if taken, err := q.UsernameTaken(ctx, "oyuncu48213"); err != nil || taken {
		t.Fatalf("UsernameTaken before insert = %v, %v", taken, err)
	}
	if n, err := q.InsertProfile(ctx, sqlc.InsertProfileParams{
		UserID: id, Username: "oyuncu48213", DisplayName: "Misafir 4821", AvatarID: "fox", CountryCode: &tr, Locale: "tr", Now: now,
	}); err != nil || n != 1 {
		t.Fatalf("InsertProfile = %d, %v", n, err)
	}
	if err := q.InsertSettings(ctx, sqlc.InsertSettingsParams{UserID: id, Now: now}); err != nil {
		t.Fatal(err)
	}
	p, err := q.GetProfile(ctx, id)
	if err != nil || p.Username != "oyuncu48213" || p.CountryCode == nil || *p.CountryCode != "TR" || p.Locale != "tr" {
		t.Fatalf("profile = %+v, %v", p, err)
	}
	s, err := q.GetSettings(ctx, id)
	if err != nil || s.ChallengesFrom != "everyone" || !s.SoundEnabled || !s.AllowFriendRequests {
		t.Fatalf("settings defaults = %+v, %v", s, err)
	}
	if taken, _ := q.UsernameTaken(ctx, "OYUNCU48213"); !taken {
		t.Fatal("username check must ignore case (citext)")
	}

	// A taken username inserts nothing (the service retries with another).
	other := ids.New()
	if _, err := pool.Exec(ctx, `INSERT INTO identity.users (id, firebase_uid, age_verified_at, last_seen_at, created_at, updated_at)
		VALUES ($1, 'fb-2', now(), now(), now(), now())`, other); err != nil {
		t.Fatal(err)
	}
	if n, err := q.InsertProfile(ctx, sqlc.InsertProfileParams{
		UserID: other, Username: "Oyuncu48213", DisplayName: "Guest 1", AvatarID: "owl", Locale: "en", Now: now,
	}); err != nil || n != 0 {
		t.Fatalf("InsertProfile with a taken username = %d, %v; want 0", n, err)
	}

	later := now.Add(time.Hour)
	p, _ = q.GetProfileForUpdate(ctx, id)
	if err := q.UpdateProfile(ctx, sqlc.UpdateProfileParams{
		UserID: id, Username: "ali.veli", DisplayName: "Ali", AvatarID: "owl", CountryCode: p.CountryCode,
		Locale: "en", UsernameChangedAt: &later, CountryChangedAt: p.CountryChangedAt, Now: later,
	}); err != nil {
		t.Fatal(err)
	}
	if p, _ = q.GetProfile(ctx, id); p.Username != "ali.veli" || p.Locale != "en" || p.UsernameChangedAt == nil {
		t.Fatalf("after UpdateProfile: %+v", p)
	}
	if err := q.UpdateSettings(ctx, sqlc.UpdateSettingsParams{
		UserID: id, ChallengesFrom: "friends", ShowOnlineStatus: false, AllowFriendRequests: true,
		SoundEnabled: false, MusicEnabled: true, HapticsEnabled: true, Now: later,
	}); err != nil {
		t.Fatal(err)
	}
	if s, _ = q.GetSettings(ctx, id); s.ChallengesFrom != "friends" || s.SoundEnabled || s.ShowOnlineStatus {
		t.Fatalf("after UpdateSettings: %+v", s)
	}
}
