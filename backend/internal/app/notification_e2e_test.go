package app

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// fakeFCM records pushes; invalid answers ErrTokenInvalid.
type fakeFCM struct {
	mu      sync.Mutex
	pushes  []notification.Push
	invalid bool
}

func (f *fakeFCM) Send(_ context.Context, p notification.Push) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.invalid {
		return "", notification.ErrTokenInvalid
	}
	f.pushes = append(f.pushes, p)
	return "msg-" + ids.NewString(), nil
}

func (f *fakeFCM) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pushes)
}

// Notifications (schema-platform §1, D-031): inbox items from events with
// de-duplication, push in the device's language, preferences, quiet
// hours, the hourly cap, invalid tokens and the purge.
func TestNotifications(t *testing.T) {
	x := newE2E(t)
	_, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	fcm := &fakeFCM{}
	notif := notification.New(notification.Deps{DB: x.db, Profiles: x.mods.profile, Sender: fcm, Clock: clock.Real{}, Logger: discard})
	inst := ids.NewString()
	want(t, "device", x.do(http.MethodPost, "/v1/me/devices", tokB, map[string]any{
		"installation_id": inst, "fcm_token": "fcm-token-b-0123456789abcdef", "platform": "android", "app_version": "1.0.0+1", "locale": "tr",
	}), 200, "")
	nameA := x.do(http.MethodGet, "/v1/profiles/"+a.String(), tokB, nil).get("display_name").(string)
	deliver := func(h events.Handler, payload any) { x.deliver(h, "test", payload) }

	// A friend request lands once (repeats collapse) and is pushed in Turkish.
	deliver(notif.OnFriendRequestSent, map[string]any{"request_id": uuid.New(), "from_user_id": a, "to_user_id": b})
	if fcm.count() != 1 || fcm.pushes[0].Title != "Arkadaşlık isteği" || fcm.pushes[0].Body != nameA+" sana arkadaşlık isteği gönderdi." ||
		fcm.pushes[0].Data["deep_link"] != "/friends" {
		t.Fatalf("pushes %+v", fcm.pushes)
	}
	r := x.do(http.MethodGet, "/v1/me/inbox", tokB, nil)
	items := r.get("items").([]any)
	if r.code != 200 || len(items) != 1 || r.get("unread") != 1.0 || items[0].(map[string]any)["title"] != "Friend request" ||
		items[0].(map[string]any)["body"] != nameA+" sent you a friend request." {
		t.Fatalf("inbox %s", r.raw)
	}
	want(t, "read nothing", x.post("/v1/me/inbox/read", tokB, map[string]any{}), 400, "VALIDATION_FAILED")
	want(t, "read", x.post("/v1/me/inbox/read", tokB, map[string]any{"ids": []any{items[0].(map[string]any)["id"]}}), 204, "")
	if r := x.do(http.MethodGet, "/v1/me/inbox", tokB, nil); r.get("unread") != 0.0 || r.get("items").([]any)[0].(map[string]any)["read"] != true {
		t.Fatalf("after read %s", r.raw)
	}

	// Preferences: defaults, a muted category, moderation always on.
	r = x.do(http.MethodGet, "/v1/me/notification-preferences", tokB, nil)
	cats := map[string]map[string]any{}
	for _, c := range r.get("categories").([]any) {
		cats[c.(map[string]any)["category"].(string)] = c.(map[string]any)
	}
	if len(cats) != 10 || cats["product_news"]["push"] != false || cats["friend"]["push"] != true || cats["moderation"]["locked"] != true {
		t.Fatalf("defaults %s", r.raw)
	}
	r = x.do(http.MethodPatch, "/v1/me/notification-preferences", tokB, map[string]any{"categories": []any{
		map[string]any{"category": "friend", "push": false, "in_app": true},
		map[string]any{"category": "moderation", "push": false, "in_app": false},
	}})
	if r.code != 200 {
		t.Fatalf("patch %s", r.raw)
	}
	want(t, "bad category", x.do(http.MethodPatch, "/v1/me/notification-preferences", tokB, map[string]any{
		"categories": []any{map[string]any{"category": "spam", "push": true, "in_app": true}}}), 400, "VALIDATION_FAILED")
	deliver(notif.OnFriendshipCreated, map[string]any{"user_id": b, "friend_id": a})
	if fcm.count() != 1 {
		t.Fatal("muted category pushed")
	}
	deliver(notif.OnSanctionApplied, map[string]any{"sanction_id": uuid.New(), "user_id": b})
	if fcm.count() != 2 || fcm.pushes[1].Data["kind"] != "moderation.sanction" {
		t.Fatalf("moderation not pushed %+v", fcm.pushes)
	}

	// Quiet hours around now (in the user's offset): inbox yes, push no.
	now := time.Now().UTC()
	start, end := now.Add(-time.Hour).Format("15:04"), now.Add(time.Hour).Format("15:04")
	want(t, "quiet", x.do(http.MethodPatch, "/v1/me/notification-preferences", tokB, map[string]any{
		"quiet_hours": map[string]any{"enabled": true, "start": start, "end": end, "utc_offset_minutes": 0}}), 200, "")
	want(t, "bad quiet", x.do(http.MethodPatch, "/v1/me/notification-preferences", tokB, map[string]any{
		"quiet_hours": map[string]any{"enabled": true, "start": "25:00", "end": end, "utc_offset_minutes": 0}}), 400, "VALIDATION_FAILED")
	challenge := func() {
		deliver(notif.OnChallengeSent, map[string]any{"challenge_id": uuid.New(), "challenger_user_id": a, "target_user_id": b})
	}
	challenge()
	if fcm.count() != 2 {
		t.Fatal("pushed in quiet hours")
	}
	var suppressed int
	if err := x.db.QueryRow(t.Context(), `SELECT count(*) FROM notification.push_deliveries WHERE status = 'suppressed'`).Scan(&suppressed); err != nil || suppressed != 1 {
		t.Fatalf("suppressed %d %v", suppressed, err)
	}

	// The hourly cap: ten pushes an hour.
	want(t, "loud", x.do(http.MethodPatch, "/v1/me/notification-preferences", tokB, map[string]any{
		"quiet_hours": map[string]any{"enabled": false, "start": start, "end": end, "utc_offset_minutes": 0}}), 200, "")
	for range 12 {
		challenge()
	}
	if fcm.count() != 10 {
		t.Fatalf("pushes in an hour %d", fcm.count())
	}

	// A token FCM rejects is forgotten.
	x.db.Exec(t.Context(), `DELETE FROM notification.push_deliveries`) //nolint:errcheck // test reset
	fcm.invalid = true
	challenge()
	var token *string
	if err := x.db.QueryRow(t.Context(), `SELECT fcm_token FROM notification.devices WHERE installation_id = $1`, inst).Scan(&token); err != nil || token != nil {
		t.Fatalf("token kept %v %v", token, err)
	}

	// Expired items go.
	if _, err := x.db.Exec(t.Context(), `UPDATE notification.inbox SET expires_at = now() - interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	if err := notif.Purge(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r := x.do(http.MethodGet, "/v1/me/inbox", tokB, nil); len(r.get("items").([]any)) != 0 {
		t.Fatalf("after purge %s", r.raw)
	}
}
