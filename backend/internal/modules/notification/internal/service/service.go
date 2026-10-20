// Package service holds notification's use cases: the in-app inbox,
// per-category preferences and quiet hours, and push delivery through FCM
// (schema-platform §1, D-031).
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// Rules (D-031).
const (
	InboxTTL      = 30 * 24 * time.Hour
	PushesPerHour = 10
	inboxPage     = 30
)

// ErrTokenInvalid is a Sender's answer for a token FCM no longer accepts.
var ErrTokenInvalid = errors.New("notification: push token invalid")

// Push is one message to one device.
type Push struct {
	Token string
	Title string
	Body  string
	// Data reaches the app with the message (kind, deep link, inbox id).
	Data map[string]string
}

// Sender delivers pushes (FCM in production).
type Sender interface {
	Send(ctx context.Context, p Push) (messageID string, err error)
}

// Pusher sends a realtime message to a user's open sessions.
type Pusher interface {
	Push(ctx context.Context, user uuid.UUID, typ string, data any) error
}

// Deps are the service's collaborators.
type Deps struct {
	DB     *pgxpool.Pool
	Sender Sender
	Pusher Pusher
	Clock  clock.Clock
	Logger *slog.Logger
}

// Service implements notification.
type Service struct{ d Deps }

// New builds the service.
func New(d Deps) *Service {
	if d.Logger == nil {
		d.Logger = slog.New(slog.DiscardHandler)
	}
	return &Service{d: d}
}

// Notice is something to tell a user.
type Notice struct {
	User     uuid.UUID
	Kind     string
	Params   map[string]string
	DeepLink string
	// Dedupe collapses repeats while unread (e.g. one per sender).
	Dedupe string
}

// Item is an inbox entry as the app shows it.
type Item struct {
	ID        uuid.UUID         `json:"id"`
	Category  string            `json:"category"`
	Kind      string            `json:"kind"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Params    map[string]string `json:"params"`
	DeepLink  *string           `json:"deep_link"`
	CreatedAt time.Time         `json:"created_at"`
	Read      bool              `json:"read"`
}

// pref is a category's channels.
type pref struct{ push, inApp bool }

func defaultPref(category string) pref { return pref{push: category != "product_news", inApp: true} }

// locked categories cannot be turned off (moderation notices).
func locked(category string) bool { return category == "moderation" }

func (s *Service) prefs(ctx context.Context, q *sqlc.Queries, user uuid.UUID) (map[string]pref, error) {
	rows, err := q.Preferences(ctx, user)
	if err != nil {
		return nil, err
	}
	out := map[string]pref{}
	for _, c := range Categories {
		out[c] = defaultPref(c)
	}
	for _, r := range rows {
		if !locked(r.Category) {
			out[r.Category] = pref{push: r.Push, inApp: r.InApp}
		}
	}
	return out, nil
}

// Notify puts a notice in the user's inbox (unless both channels are off
// for its category), tells the open app and pushes it to the user's
// devices when allowed: push on, outside quiet hours, under 10 an hour.
// A repeat of an unread notice with the same dedupe key only refreshes it.
func (s *Service) Notify(ctx context.Context, n Notice) error {
	t, ok := templates[n.Kind]
	if !ok {
		return fmt.Errorf("notification: unknown kind %q", n.Kind)
	}
	q := sqlc.New(s.d.DB)
	prefs, err := s.prefs(ctx, q, n.User)
	if err != nil {
		return err
	}
	p := prefs[t.category]
	if !p.push && !p.inApp {
		return nil
	}
	params, err := json.Marshal(n.Params)
	if err != nil {
		return err
	}
	now := s.d.Clock.Now()
	row, err := q.InsertInbox(ctx, sqlc.InsertInboxParams{
		ID: ids.New(), UserID: n.User, Category: t.category, Kind: n.Kind, Params: params,
		DeepLink: optional(n.DeepLink), DedupeKey: optional(n.Dedupe), Now: now, ExpiresAt: now.Add(InboxTTL),
	})
	if err != nil {
		return err
	}
	if !row.Inserted {
		return nil
	}
	if s.d.Pusher != nil {
		if err := s.d.Pusher.Push(ctx, n.User, "inbox.new", map[string]any{"id": row.ID, "kind": n.Kind}); err != nil {
			s.d.Logger.DebugContext(ctx, "notification: inbox.new not delivered", "error", err)
		}
	}
	if !p.push || s.d.Sender == nil {
		return nil
	}
	return s.push(ctx, q, row.ID, n, now)
}

func optional(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// push sends an inbox item to every device with a token, or records why
// it was held back (quiet hours, hourly cap).
func (s *Service) push(ctx context.Context, q *sqlc.Queries, inbox uuid.UUID, n Notice, now time.Time) error {
	targets, err := q.PushTargets(ctx, n.User)
	if err != nil || len(targets) == 0 {
		return err
	}
	suppress := ""
	quiet, err := s.inQuietHours(ctx, q, n.User, now)
	if err != nil {
		return err
	}
	if quiet {
		suppress = "quiet_hours"
	} else {
		sent, err := q.PushesSince(ctx, sqlc.PushesSinceParams{UserID: n.User, Since: now.Add(-time.Hour)})
		if err != nil {
			return err
		}
		if sent >= PushesPerHour {
			suppress = "hourly_cap"
		}
	}
	for _, d := range targets {
		status, msgID, failure := "suppressed", "", suppress
		if suppress == "" {
			title, body := render(n.Kind, d.Locale, n.Params)
			msgID, err = s.d.Sender.Send(ctx, Push{Token: *d.FcmToken, Title: title, Body: body, Data: map[string]string{
				"kind": n.Kind, "inbox_id": inbox.String(), "deep_link": n.DeepLink,
			}})
			switch {
			case errors.Is(err, ErrTokenInvalid):
				status, failure = "token_invalid", err.Error()
				if err := q.RevokeToken(ctx, d.ID); err != nil {
					return err
				}
			case err != nil:
				status, failure = "failed", err.Error()
				s.d.Logger.WarnContext(ctx, "notification: push failed", "user_id", n.User, "error", err)
			default:
				status, failure = "sent", ""
			}
		}
		if err := q.InsertDelivery(ctx, sqlc.InsertDeliveryParams{
			ID: ids.New(), InboxID: inbox, DeviceID: d.ID, Status: status, FcmMessageID: optional(msgID),
			Error: optional(failure), Now: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func minutes(t pgtype.Time) int { return int(t.Microseconds / 60_000_000) }

func clockTime(m int) pgtype.Time {
	return pgtype.Time{Microseconds: int64(m) * 60_000_000, Valid: true}
}

// inQuietHours reports whether now falls in the user's quiet hours, in
// the user's local time (the window may wrap past midnight).
func (s *Service) inQuietHours(ctx context.Context, q *sqlc.Queries, user uuid.UUID, now time.Time) (bool, error) {
	qh, err := q.QuietHours(ctx, user)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil || !qh.Enabled {
		return false, err
	}
	local := now.UTC().Add(time.Duration(qh.UtcOffsetMinutes) * time.Minute)
	m := local.Hour()*60 + local.Minute()
	start, end := minutes(qh.StartLocal), minutes(qh.EndLocal)
	if start <= end {
		return m >= start && m < end, nil
	}
	return m >= start || m < end, nil
}

// Inbox is GET /v1/me/inbox.
type Inbox struct {
	Items      []Item  `json:"items"`
	Unread     int     `json:"unread"`
	NextCursor *string `json:"next_cursor"`
}

// List returns a page of the user's inbox, newest first, rendered in
// locale; cursor is the previous page's last created_at.
func (s *Service) List(ctx context.Context, user uuid.UUID, locale string, cursor *time.Time) (Inbox, error) {
	q := sqlc.New(s.d.DB)
	now := s.d.Clock.Now()
	rows, err := q.Inbox(ctx, sqlc.InboxParams{UserID: user, Now: now, Before: cursor, Lim: inboxPage + 1})
	if err != nil {
		return Inbox{}, err
	}
	unread, err := q.UnreadCount(ctx, sqlc.UnreadCountParams{UserID: user, Now: now})
	if err != nil {
		return Inbox{}, err
	}
	out := Inbox{Items: []Item{}, Unread: int(unread)}
	for i := range rows {
		if i == inboxPage {
			next := rows[i-1].CreatedAt.Format(time.RFC3339Nano)
			out.NextCursor = &next
			break
		}
		r := &rows[i]
		params := map[string]string{}
		if err := json.Unmarshal(r.Params, &params); err != nil {
			return Inbox{}, err
		}
		title, body := render(r.Kind, locale, params)
		out.Items = append(out.Items, Item{
			ID: r.ID, Category: r.Category, Kind: r.Kind, Title: title, Body: body, Params: params,
			DeepLink: r.DeepLink, CreatedAt: r.CreatedAt, Read: r.ReadAt != nil,
		})
	}
	return out, nil
}

// MarkRead marks some (or all) of the user's items read.
func (s *Service) MarkRead(ctx context.Context, user uuid.UUID, items []uuid.UUID, all bool) error {
	if !all && len(items) == 0 {
		return apierror.Validation(apierror.FieldError{Field: "ids", Rule: "required"})
	}
	now := s.d.Clock.Now()
	return sqlc.New(s.d.DB).MarkRead(ctx, sqlc.MarkReadParams{Now: &now, UserID: user, Every: all, Ids: items})
}

// CategoryPref is one category's channels.
type CategoryPref struct {
	Category string `json:"category"`
	Push     bool   `json:"push"`
	InApp    bool   `json:"in_app"`
	// Locked categories cannot be turned off (moderation notices).
	Locked bool `json:"locked"`
}

// QuietHours are local times "HH:MM" and the device's UTC offset.
type QuietHours struct {
	Enabled          bool   `json:"enabled"`
	Start            string `json:"start"`
	End              string `json:"end"`
	UTCOffsetMinutes int    `json:"utc_offset_minutes"`
}

// Preferences is GET/PATCH /v1/me/notification-preferences.
type Preferences struct {
	Categories []CategoryPref `json:"categories"`
	QuietHours QuietHours     `json:"quiet_hours"`
}

// PreferencesInput is the PATCH body: only the given parts change.
type PreferencesInput struct {
	Categories []CategoryPref `json:"categories,omitempty"`
	QuietHours *QuietHours    `json:"quiet_hours,omitempty"`
}

// Preferences returns the user's settings with the defaults filled in.
func (s *Service) Preferences(ctx context.Context, user uuid.UUID) (Preferences, error) {
	q := sqlc.New(s.d.DB)
	prefs, err := s.prefs(ctx, q, user)
	if err != nil {
		return Preferences{}, err
	}
	out := Preferences{QuietHours: QuietHours{Start: "23:00", End: "08:00"}}
	for _, c := range Categories {
		out.Categories = append(out.Categories, CategoryPref{Category: c, Push: prefs[c].push, InApp: prefs[c].inApp, Locked: locked(c)})
	}
	qh, err := q.QuietHours(ctx, user)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return Preferences{}, err
	default:
		out.QuietHours = QuietHours{
			Enabled: qh.Enabled, Start: hhmm(minutes(qh.StartLocal)), End: hhmm(minutes(qh.EndLocal)),
			UTCOffsetMinutes: int(qh.UtcOffsetMinutes),
		}
	}
	return out, nil
}

func hhmm(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

func parseHHMM(v string) (int, bool) {
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

// UpdatePreferences changes categories and quiet hours.
func (s *Service) UpdatePreferences(ctx context.Context, user uuid.UUID, in PreferencesInput) (Preferences, error) {
	for _, c := range in.Categories {
		if !slices.Contains(Categories, c.Category) {
			return Preferences{}, apierror.Validation(apierror.FieldError{Field: "categories.category", Rule: "one_of"})
		}
	}
	var start, end int
	if qh := in.QuietHours; qh != nil {
		var ok1, ok2 bool
		start, ok1 = parseHHMM(qh.Start)
		end, ok2 = parseHHMM(qh.End)
		switch {
		case !ok1 || !ok2:
			return Preferences{}, apierror.Validation(apierror.FieldError{Field: "quiet_hours", Rule: "format"})
		case qh.UTCOffsetMinutes < -14*60 || qh.UTCOffsetMinutes > 14*60:
			return Preferences{}, apierror.Validation(apierror.FieldError{Field: "quiet_hours.utc_offset_minutes", Rule: "range"})
		}
	}
	q := sqlc.New(s.d.DB)
	now := s.d.Clock.Now()
	for _, c := range in.Categories {
		if locked(c.Category) {
			continue
		}
		if err := q.UpsertPreference(ctx, sqlc.UpsertPreferenceParams{UserID: user, Category: c.Category, Push: c.Push, InApp: c.InApp, Now: now}); err != nil {
			return Preferences{}, err
		}
	}
	if qh := in.QuietHours; qh != nil {
		if err := q.UpsertQuietHours(ctx, sqlc.UpsertQuietHoursParams{
			UserID: user, Enabled: qh.Enabled, StartLocal: clockTime(start), EndLocal: clockTime(end),
			UtcOffsetMinutes: int16(qh.UTCOffsetMinutes), //nolint:gosec // G115: checked ±840
		}); err != nil {
			return Preferences{}, err
		}
	}
	return s.Preferences(ctx, user)
}

// Purge deletes expired inbox items and deliveries older than 30 days.
func (s *Service) Purge(ctx context.Context) error {
	q := sqlc.New(s.d.DB)
	now := s.d.Clock.Now()
	if _, err := q.PurgeInbox(ctx, now); err != nil {
		return err
	}
	_, err := q.PurgeDeliveries(ctx, now.Add(-InboxTTL))
	return err
}
