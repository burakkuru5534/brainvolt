// Package service holds chat's use cases: channels of persistent teams
// and couples, sending with the word filter, mutes and the rate limit,
// history, removal and read cursors (schema-platform §2, D-031, D-088).
package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/chat/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// Rules (D-031).
const (
	MaxBody       = 300
	BurstMessages = 5
	BurstWindow   = 10 * time.Second
	Retention     = 183 * 24 * time.Hour
	page          = 50
)

// QuickPhrases are the preset messages (translated on the device).
var QuickPhrases = []string{"good_luck", "well_played", "nice", "thanks", "oops", "lets_go", "gg", "rematch"}

// Teams reads team and couple membership (team).
type Teams interface {
	TeamInfo(ctx context.Context, id uuid.UUID) (team.Info, error)
	CoupleInfo(ctx context.Context, id uuid.UUID) (team.CoupleInfo, error)
}

// Cards reads senders' public cards (profile).
type Cards interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
}

// Muter reports whether a user may not chat (moderation's chat_mute or
// suspension; wired in app, chat does not import moderation).
type Muter interface {
	Muted(ctx context.Context, user uuid.UUID) (bool, error)
}

// Realtime delivers chat events to members' sessions.
type Realtime interface {
	Send(ctx context.Context, t realtime.Target, f realtime.Frame) error
}

// Deps are the service's collaborators.
type Deps struct {
	DB    *pgxpool.Pool
	Teams Teams
	Cards Cards
	Muter Muter
	RT    Realtime
	Clock clock.Clock
}

// Service implements chat.
type Service struct{ d Deps }

// New builds the service.
func New(d Deps) *Service { return &Service{d: d} }

// Channel is a channel as its members see it.
type Channel struct {
	ID       uuid.UUID      `json:"id"`
	Kind     string         `json:"kind"`
	TeamID   *uuid.UUID     `json:"team_id,omitempty"`
	CoupleID *uuid.UUID     `json:"couple_id,omitempty"`
	Members  []profile.Card `json:"members"`
	Unread   int            `json:"unread"`
}

// Message is a chat message; removed ones keep their place without body.
type Message struct {
	ID        uuid.UUID    `json:"id"`
	ChannelID uuid.UUID    `json:"channel_id"`
	Sender    profile.Card `json:"sender"`
	Body      *string      `json:"body,omitempty"`
	QuickID   *string      `json:"quick_id,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	Removed   bool         `json:"removed"`
}

// OpenInput names the team or the couple whose channel to open.
type OpenInput struct {
	TeamID   *uuid.UUID `json:"team_id,omitempty"`
	CoupleID *uuid.UUID `json:"couple_id,omitempty"`
}

func notFound() error { return apierror.New(apierror.NotFound) }

// members returns a channel's current members, or NOT_FOUND when user is
// not one of them (or the team / couple is gone).
func (s *Service) members(ctx context.Context, c *sqlc.ChatChannel, user uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	switch c.Kind {
	case "team":
		t, err := s.d.Teams.TeamInfo(ctx, *c.TeamID)
		if err != nil || !t.Active {
			return nil, notFound()
		}
		out = t.Members
	case "couple":
		cp, err := s.d.Teams.CoupleInfo(ctx, *c.CoupleID)
		if err != nil || !cp.Active {
			return nil, notFound()
		}
		out = cp.Users[:]
	default:
		return nil, notFound()
	}
	if !slices.Contains(out, user) {
		return nil, notFound()
	}
	return out, nil
}

func (s *Service) channel(ctx context.Context, user, id uuid.UUID) (*sqlc.ChatChannel, []uuid.UUID, error) {
	c, err := sqlc.New(s.d.DB).GetChannel(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, notFound()
	}
	if err != nil {
		return nil, nil, err
	}
	members, err := s.members(ctx, &c, user)
	return &c, members, err
}

// Open returns the channel of the caller's team or couple, creating it
// on first use.
func (s *Service) Open(ctx context.Context, user uuid.UUID, in OpenInput) (Channel, error) {
	if (in.TeamID == nil) == (in.CoupleID == nil) {
		return Channel{}, apierror.Validation(apierror.FieldError{Field: "team_id", Rule: "one_of"})
	}
	q := sqlc.New(s.d.DB)
	now := s.d.Clock.Now()
	var c sqlc.ChatChannel
	var err error
	if in.TeamID != nil {
		c, err = q.TeamChannel(ctx, sqlc.TeamChannelParams{ID: ids.New(), TeamID: in.TeamID, Now: now})
	} else {
		c, err = q.CoupleChannel(ctx, sqlc.CoupleChannelParams{ID: ids.New(), CoupleID: in.CoupleID, Now: now})
	}
	if err != nil {
		return Channel{}, err
	}
	members, err := s.members(ctx, &c, user)
	if err != nil {
		return Channel{}, err
	}
	cards, err := s.d.Cards.Cards(ctx, members)
	if err != nil {
		return Channel{}, err
	}
	unread, err := q.Unread(ctx, sqlc.UnreadParams{ChannelID: c.ID, UserID: user})
	if err != nil {
		return Channel{}, err
	}
	out := Channel{ID: c.ID, Kind: c.Kind, TeamID: c.TeamID, CoupleID: c.CoupleID, Members: []profile.Card{}, Unread: int(unread)}
	for _, m := range members {
		out.Members = append(out.Members, cards[m])
	}
	return out, nil
}

// SendInput is chat.send: free text or a preset phrase.
type SendInput struct {
	ChannelID uuid.UUID `json:"channel_id"`
	Body      *string   `json:"body,omitempty"`
	QuickID   *string   `json:"quick_id,omitempty"`
}

// Send posts a message: members only, not while muted, 5 per 10 s; text
// is trimmed to 1–300 characters, blocked words are masked and slurs
// refused. Every member's sessions get chat.message.
func (s *Service) Send(ctx context.Context, user uuid.UUID, in SendInput) (Message, error) {
	kind, filter := "text", "clean"
	switch {
	case (in.Body == nil) == (in.QuickID == nil):
		return Message{}, apierror.Validation(apierror.FieldError{Field: "body", Rule: "one_of"})
	case in.QuickID != nil:
		if !slices.Contains(QuickPhrases, *in.QuickID) {
			return Message{}, apierror.Validation(apierror.FieldError{Field: "quick_id", Rule: "one_of"})
		}
		kind = "quick"
	default:
		body := strings.TrimSpace(*in.Body)
		if n := len([]rune(body)); n == 0 || n > MaxBody {
			return Message{}, apierror.Validation(apierror.FieldError{Field: "body", Rule: "length"})
		}
		masked, changed, severe := profile.MaskText(body)
		if severe {
			return Message{}, apierror.New(apierror.TextRejected)
		}
		if changed {
			filter = "masked"
		}
		in.Body = &masked
	}
	c, members, err := s.channel(ctx, user, in.ChannelID)
	if err != nil {
		return Message{}, err
	}
	if s.d.Muter != nil {
		muted, err := s.d.Muter.Muted(ctx, user)
		if err != nil {
			return Message{}, err
		}
		if muted {
			return Message{}, apierror.New(apierror.Sanctioned).With("kind", "chat_mute")
		}
	}
	q := sqlc.New(s.d.DB)
	now := s.d.Clock.Now()
	sent, err := q.SentSince(ctx, sqlc.SentSinceParams{SenderID: user, Since: now.Add(-BurstWindow)})
	if err != nil {
		return Message{}, err
	}
	if sent >= BurstMessages {
		return Message{}, apierror.New(apierror.RateLimited).WithRetryAfter(BurstWindow)
	}
	id := ids.New()
	if err := q.InsertMessage(ctx, sqlc.InsertMessageParams{
		ID: id, ChannelID: c.ID, SenderID: user, Kind: kind, Body: in.Body, QuickID: in.QuickID, FilterResult: filter, Now: now,
	}); err != nil {
		return Message{}, err
	}
	cards, err := s.d.Cards.Cards(ctx, []uuid.UUID{user})
	if err != nil {
		return Message{}, err
	}
	m := Message{ID: id, ChannelID: c.ID, Sender: cards[user], Body: in.Body, QuickID: in.QuickID, CreatedAt: now}
	s.broadcast(ctx, members, realtime.Frame{T: "chat.message", S: stream(c.ID), TS: now.UnixMilli(), D: map[string]any{"message": m}})
	return m, nil
}

func stream(id uuid.UUID) string { return "chat:" + id.String() }

func (s *Service) broadcast(ctx context.Context, users []uuid.UUID, f realtime.Frame) {
	if s.d.RT == nil {
		return
	}
	for _, u := range users {
		_ = s.d.RT.Send(ctx, realtime.Target{User: u}, f) // best effort: history is the source of truth
	}
}

// Typing tells the other members someone is typing.
func (s *Service) Typing(ctx context.Context, user, channel uuid.UUID) error {
	_, members, err := s.channel(ctx, user, channel)
	if err != nil {
		return err
	}
	others := slices.DeleteFunc(slices.Clone(members), func(u uuid.UUID) bool { return u == user })
	s.broadcast(ctx, others, realtime.Frame{T: "chat.typing", S: stream(channel), D: map[string]any{"channel_id": channel, "user_id": user}})
	return nil
}

// History is GET /v1/chat/channels/{id}/messages.
type History struct {
	Messages   []Message `json:"messages"`
	NextCursor *string   `json:"next_cursor"`
}

// History returns messages newest first; cursor is a message id.
func (s *Service) History(ctx context.Context, user, channel uuid.UUID, before *uuid.UUID) (History, error) {
	if _, _, err := s.channel(ctx, user, channel); err != nil {
		return History{}, err
	}
	rows, err := sqlc.New(s.d.DB).History(ctx, sqlc.HistoryParams{ChannelID: channel, Before: before, Lim: page + 1})
	if err != nil {
		return History{}, err
	}
	senders := []uuid.UUID{}
	for i := range rows {
		senders = append(senders, rows[i].SenderID)
	}
	cards, err := s.d.Cards.Cards(ctx, slices.Compact(senders))
	if err != nil {
		return History{}, err
	}
	out := History{Messages: []Message{}}
	for i := range rows {
		if i == page {
			next := rows[i-1].ID.String()
			out.NextCursor = &next
			break
		}
		r := &rows[i]
		out.Messages = append(out.Messages, Message{
			ID: r.ID, ChannelID: r.ChannelID, Sender: cards[r.SenderID], Body: r.Body, QuickID: r.QuickID,
			CreatedAt: r.CreatedAt, Removed: r.RemovedAt != nil,
		})
	}
	return out, nil
}

// Remove blanks a message: its sender, or staff.
func (s *Service) Remove(ctx context.Context, user, id uuid.UUID, staff bool) error {
	q := sqlc.New(s.d.DB)
	m, err := q.GetMessage(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return err
	}
	var members []uuid.UUID
	c, err := q.GetChannel(ctx, m.ChannelID)
	if err != nil {
		return err
	}
	if !staff {
		if m.SenderID != user {
			return notFound()
		}
		if members, err = s.members(ctx, &c, user); err != nil {
			return err
		}
	}
	now := s.d.Clock.Now()
	if _, err := q.RemoveMessage(ctx, sqlc.RemoveMessageParams{ID: id, Now: &now, RemovedBy: &user}); err != nil {
		return err
	}
	s.broadcast(ctx, members, realtime.Frame{T: "chat.message_removed", S: stream(c.ID), D: map[string]any{"channel_id": c.ID, "id": id}})
	return nil
}

// Read moves the caller's read cursor forward.
func (s *Service) Read(ctx context.Context, user, channel, last uuid.UUID) error {
	if _, _, err := s.channel(ctx, user, channel); err != nil {
		return err
	}
	return sqlc.New(s.d.DB).UpsertReadCursor(ctx, sqlc.UpsertReadCursorParams{ChannelID: channel, UserID: user, LastReadID: last})
}

// Purge deletes messages older than six months.
func (s *Service) Purge(ctx context.Context) error {
	_, err := sqlc.New(s.d.DB).PurgeMessages(ctx, s.d.Clock.Now().Add(-Retention))
	return err
}
