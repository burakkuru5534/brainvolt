// Package service implements the team module (schema-social.md §4):
// persistent teams with memberships and invites, and couple profiles
// (D-018, D-047), with the limits of D-027. Only full accounts take part
// (D-016a).
package service

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Limits (D-027).
const (
	MinMembers     = 2
	MaxMembers     = 4
	TeamsPerUser   = 3
	InviteTTL      = 7 * 24 * time.Hour
	CoupleTTL      = 7 * 24 * time.Hour
	CoupleCooldown = 7 * 24 * time.Hour
)

// Event names (schema-social.md §5).
const (
	EventTeamCreated     = "team.TeamCreated"
	EventTeamDisbanded   = "team.TeamDisbanded"
	EventMemberJoined    = "team.MemberJoined"
	EventMemberLeft      = "team.MemberLeft"
	EventCaptainChanged  = "team.CaptainChanged"
	EventCoupleRequested = "team.CoupleRequested"
	EventCoupleFormed    = "team.CoupleFormed"
	EventCoupleDissolved = "team.CoupleDissolved"
)

// Emblems are the built-in team emblems (D-069).
var Emblems = []string{"bolt", "flame", "shield", "star", "crown", "rocket", "brain", "owl", "lion", "wave"}

var tagPattern = regexp.MustCompile(`^[A-Za-z0-9]{2,4}$`)

// Users is what team needs from identity.
type Users interface {
	User(ctx context.Context, id uuid.UUID) (identity.User, error)
}

// Profiles is what team needs from profile.
type Profiles interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
}

// Social is what team needs from social.
type Social interface {
	AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error)
	IsBlocked(ctx context.Context, a, b uuid.UUID) (bool, error)
}

// Deps are the service's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Users    Users
	Profiles Profiles
	Social   Social
	Events   *events.Bus
	Clock    clock.Clock
}

// Service implements team's use cases.
type Service struct{ d Deps }

// New returns the service.
func New(d Deps) *Service { return &Service{d: d} }

// Event payloads.
type (
	// TeamEvent is the payload of TeamCreated / TeamDisbanded.
	TeamEvent struct {
		TeamID uuid.UUID `json:"team_id"`
	}
	// MemberEvent is the payload of MemberJoined / MemberLeft / CaptainChanged.
	MemberEvent struct {
		TeamID uuid.UUID `json:"team_id"`
		UserID uuid.UUID `json:"user_id"`
		Reason string    `json:"reason,omitempty"`
	}
	// CoupleEvent is the payload of the couple events.
	CoupleEvent struct {
		CoupleID uuid.UUID    `json:"couple_id"`
		UserIDs  [2]uuid.UUID `json:"user_ids"`
	}
)

func (s *Service) now() time.Time { return s.d.Clock.Now().UTC() }

// fullUser fails unless id is an active full account.
func (s *Service) fullUser(ctx context.Context, id uuid.UUID, feature string) error {
	u, err := s.d.Users.User(ctx, id)
	if err != nil {
		return err
	}
	if u.Status != identity.StatusActive || u.DeletionRequestedAt != nil {
		return apierror.New(apierror.NotFound)
	}
	if u.Tier != identity.TierFull {
		return apierror.New(apierror.AccountUpgradeRequired).With("feature", feature)
	}
	return nil
}

// Team is a team profile with its members.
type Team struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Tag           string    `json:"tag"`
	EmblemID      string    `json:"emblem_id"`
	CaptainUserID uuid.UUID `json:"captain_user_id"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	Members       []Member  `json:"members"`
}

// Member is a team member with their card.
type Member struct {
	profile.Card
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// TeamInput is the body of POST /v1/teams and PATCH /v1/teams/{id}.
type TeamInput struct {
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	EmblemID string `json:"emblem_id"`
}

func (in *TeamInput) validate() error {
	in.Name, in.Tag = strings.Join(strings.Fields(in.Name), " "), strings.ToUpper(strings.TrimSpace(in.Tag))
	var f apierror.Fields
	if n := len([]rune(in.Name)); n < 3 || n > 24 {
		f.Add("name", "length", "min", 3, "max", 24)
	}
	if !tagPattern.MatchString(in.Tag) {
		f.Add("tag", "format")
	}
	if !slices.Contains(Emblems, in.EmblemID) {
		f.Add("emblem_id", "enum", "allowed", Emblems)
	}
	if err := f.Err(); err != nil {
		return err
	}
	if profile.TextRejected(in.Name) || profile.TextRejected(in.Tag) {
		return apierror.New(apierror.TextRejected)
	}
	return nil
}

// checkNames fails when an active team already uses the name or tag.
func checkNames(ctx context.Context, q *sqlc.Queries, in TeamInput, self uuid.UUID) error {
	r, err := q.NameOrTagTaken(ctx, sqlc.NameOrTagTakenParams{Name: in.Name, Tag: in.Tag, Self: self})
	if err != nil {
		return err
	}
	var f apierror.Fields
	if r.NameTaken {
		f.Add("name", "taken")
	}
	if r.TagTaken {
		f.Add("tag", "taken")
	}
	return f.Err()
}

// uniqueViolation maps a race on the active name/tag indexes.
func uniqueViolation(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		field := "name"
		if strings.Contains(pe.ConstraintName, "tag") {
			field = "tag"
		}
		return apierror.Validation(apierror.FieldError{Field: field, Rule: "taken"})
	}
	return err
}

// CreateTeam creates a team with the caller as captain (max 3 active teams
// per user).
func (s *Service) CreateTeam(ctx context.Context, user uuid.UUID, in TeamInput) (Team, error) {
	if err := s.fullUser(ctx, user, "teams"); err != nil {
		return Team{}, err
	}
	if err := in.validate(); err != nil {
		return Team{}, err
	}
	id, now := ids.New(), s.now()
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := s.teamLimit(ctx, q, user); err != nil {
			return err
		}
		if err := checkNames(ctx, q, in, uuid.Nil); err != nil {
			return err
		}
		if err := q.InsertTeam(ctx, sqlc.InsertTeamParams{ID: id, Name: in.Name, Tag: in.Tag, EmblemID: in.EmblemID, CaptainUserID: user, Now: now}); err != nil {
			return uniqueViolation(err)
		}
		if err := q.InsertMembership(ctx, sqlc.InsertMembershipParams{TeamID: id, UserID: user, Role: "captain", Now: now}); err != nil {
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventTeamCreated, now, TeamEvent{TeamID: id})
	})
	if err != nil {
		return Team{}, err
	}
	return s.Team(ctx, id)
}

func (s *Service) teamLimit(ctx context.Context, q *sqlc.Queries, user uuid.UUID) error {
	n, err := q.ActiveTeamCount(ctx, user)
	if err != nil {
		return err
	}
	if n >= TeamsPerUser {
		return apierror.New(apierror.TeamLimitReached).With("limit", TeamsPerUser)
	}
	return nil
}

// Team returns a team with its current members.
func (s *Service) Team(ctx context.Context, id uuid.UUID) (Team, error) {
	q := sqlc.New(s.d.DB)
	t, err := q.GetTeam(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Team{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Team{}, err
	}
	ms, err := q.Members(ctx, id)
	if err != nil {
		return Team{}, err
	}
	users := make([]uuid.UUID, len(ms))
	for i, m := range ms {
		users[i] = m.UserID
	}
	cards, err := s.d.Profiles.Cards(ctx, users)
	if err != nil {
		return Team{}, err
	}
	out := Team{ID: t.ID, Name: t.Name, Tag: t.Tag, EmblemID: t.EmblemID, CaptainUserID: t.CaptainUserID, Status: t.Status,
		CreatedAt: t.CreatedAt, Members: make([]Member, 0, len(ms))}
	for _, m := range ms {
		out.Members = append(out.Members, Member{Card: cards[m.UserID], Role: m.Role, JoinedAt: m.JoinedAt})
	}
	return out, nil
}

// MyTeams lists the caller's active teams.
func (s *Service) MyTeams(ctx context.Context, user uuid.UUID) ([]Team, error) {
	teamIDs, err := sqlc.New(s.d.DB).UserTeamIDs(ctx, user)
	if err != nil {
		return nil, err
	}
	out := make([]Team, 0, len(teamIDs))
	for _, id := range teamIDs {
		t, err := s.Team(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// locked loads an active team for a change and the caller's role in it.
func locked(ctx context.Context, q *sqlc.Queries, id, user uuid.UUID) ([]sqlc.MembersRow, string, error) {
	t, err := q.GetTeamForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", apierror.New(apierror.NotFound)
	}
	if err != nil {
		return nil, "", err
	}
	if t.Status != "active" {
		return nil, "", apierror.New(apierror.NotFound).With("reason", "team_disbanded")
	}
	ms, err := q.Members(ctx, id)
	if err != nil {
		return nil, "", err
	}
	for _, m := range ms {
		if m.UserID == user {
			return ms, m.Role, nil
		}
	}
	return ms, "", apierror.New(apierror.NotFound)
}

// UpdateTeam changes name, tag or emblem (captain).
func (s *Service) UpdateTeam(ctx context.Context, user, id uuid.UUID, in TeamInput) (Team, error) {
	if err := in.validate(); err != nil {
		return Team{}, err
	}
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		_, role, err := locked(ctx, q, id, user)
		if err != nil {
			return err
		}
		if role != "captain" {
			return apierror.New(apierror.NotCaptain)
		}
		if err := checkNames(ctx, q, in, id); err != nil {
			return err
		}
		return uniqueViolation(q.UpdateTeam(ctx, sqlc.UpdateTeamParams{ID: id, Name: in.Name, Tag: in.Tag, EmblemID: in.EmblemID, Now: s.now()}))
	})
	if err != nil {
		return Team{}, err
	}
	return s.Team(ctx, id)
}

// Invite is a pending team invite as the invited user sees it.
type Invite struct {
	ID        uuid.UUID    `json:"id"`
	TeamID    uuid.UUID    `json:"team_id"`
	TeamName  string       `json:"team_name"`
	TeamTag   string       `json:"team_tag"`
	EmblemID  string       `json:"emblem_id"`
	InvitedBy profile.Card `json:"invited_by"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// InviteToTeam invites a friend of the caller (any member may invite) to
// a team with room left; invites last 7 days.
func (s *Service) InviteToTeam(ctx context.Context, user, id, invitee uuid.UUID) (uuid.UUID, error) {
	if user == invitee {
		return uuid.Nil, apierror.Validation(apierror.FieldError{Field: "user_id", Rule: "not_self"})
	}
	if err := s.fullUser(ctx, invitee, "teams"); err != nil {
		return uuid.Nil, err
	}
	friends, err := s.d.Social.AreFriends(ctx, user, invitee)
	if err != nil {
		return uuid.Nil, err
	}
	if !friends {
		return uuid.Nil, apierror.New(apierror.Forbidden).With("reason", "not_friends")
	}
	inviteID, now := ids.New(), s.now()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		ms, _, err := locked(ctx, q, id, user)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(ms, func(m sqlc.MembersRow) bool { return m.UserID == invitee }) {
			return apierror.Validation(apierror.FieldError{Field: "user_id", Rule: "already_member"})
		}
		if len(ms) >= MaxMembers {
			return apierror.New(apierror.TeamFull).With("limit", MaxMembers)
		}
		if err := q.ExpireInvites(ctx, sqlc.ExpireInvitesParams{TeamID: id, Now: now}); err != nil {
			return err
		}
		pending, err := q.PendingInvite(ctx, sqlc.PendingInviteParams{TeamID: id, UserID: invitee, Now: now})
		if err != nil {
			return err
		}
		if pending {
			return apierror.New(apierror.RequestPending)
		}
		return q.InsertInvite(ctx, sqlc.InsertInviteParams{ID: inviteID, TeamID: id, InvitedUserID: invitee, InvitedBy: user, Now: now, ExpiresAt: now.Add(InviteTTL)})
	})
	return inviteID, err
}

// Invites lists the caller's pending team invites.
func (s *Service) Invites(ctx context.Context, user uuid.UUID) ([]Invite, error) {
	rows, err := sqlc.New(s.d.DB).InvitesOf(ctx, sqlc.InvitesOfParams{UserID: user, Now: s.now()})
	if err != nil {
		return nil, err
	}
	by := make([]uuid.UUID, len(rows))
	for i := range rows {
		by[i] = rows[i].InvitedBy
	}
	cards, err := s.d.Profiles.Cards(ctx, by)
	if err != nil {
		return nil, err
	}
	out := make([]Invite, len(rows))
	for i := range rows {
		r := &rows[i]
		out[i] = Invite{ID: r.ID, TeamID: r.TeamID, TeamName: r.TeamName, TeamTag: r.TeamTag, EmblemID: r.TeamEmblemID,
			InvitedBy: cards[r.InvitedBy], CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt}
	}
	return out, nil
}

// RespondInvite accepts or declines an invite (the invited user).
func (s *Service) RespondInvite(ctx context.Context, user, id uuid.UUID, accept bool) error {
	if accept {
		if err := s.fullUser(ctx, user, "teams"); err != nil {
			return err
		}
	}
	now := s.now()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		inv, err := q.GetInviteForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && inv.InvitedUserID != user) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		if inv.Status != "pending" {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "invite_"+inv.Status)
		}
		if !inv.ExpiresAt.After(now) {
			if err := q.SetInviteStatus(ctx, sqlc.SetInviteStatusParams{ID: id, Status: "expired"}); err != nil {
				return err
			}
			return apierror.New(apierror.ChallengeExpired).With("reason", "invite_expired")
		}
		if !accept {
			return q.SetInviteStatus(ctx, sqlc.SetInviteStatusParams{ID: id, Status: "declined", RespondedAt: &now})
		}
		t, err := q.GetTeamForUpdate(ctx, inv.TeamID)
		if err != nil {
			return err
		}
		if t.Status != "active" {
			return apierror.New(apierror.NotFound).With("reason", "team_disbanded")
		}
		ms, err := q.Members(ctx, inv.TeamID)
		if err != nil {
			return err
		}
		if len(ms) >= MaxMembers {
			return apierror.New(apierror.TeamFull).With("limit", MaxMembers)
		}
		if err := s.teamLimit(ctx, q, user); err != nil {
			return err
		}
		for _, m := range ms {
			blocked, err := s.d.Social.IsBlocked(ctx, user, m.UserID)
			if err != nil {
				return err
			}
			if blocked {
				return apierror.New(apierror.Blocked)
			}
		}
		if err := q.InsertMembership(ctx, sqlc.InsertMembershipParams{TeamID: inv.TeamID, UserID: user, Role: "member", Now: now}); err != nil {
			return err
		}
		if err := q.SetInviteStatus(ctx, sqlc.SetInviteStatusParams{ID: id, Status: "accepted", RespondedAt: &now}); err != nil {
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventMemberJoined, now, MemberEvent{TeamID: inv.TeamID, UserID: user})
	})
}

// LeaveTeam removes the caller. A leaving captain hands over to the
// longest-standing member; the last member leaving disbands the team.
func (s *Service) LeaveTeam(ctx context.Context, user, id uuid.UUID) error {
	return s.removeMember(ctx, user, id, user, "left")
}

// Kick removes another member (captain).
func (s *Service) Kick(ctx context.Context, captain, id, member uuid.UUID) error {
	if captain == member {
		return apierror.Validation(apierror.FieldError{Field: "user_id", Rule: "not_self"})
	}
	return s.removeMember(ctx, captain, id, member, "kicked")
}

func (s *Service) removeMember(ctx context.Context, actor, id, member uuid.UUID, reason string) error {
	now := s.now()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		ms, role, err := locked(ctx, q, id, actor)
		if err != nil {
			return err
		}
		if actor != member && role != "captain" {
			return apierror.New(apierror.NotCaptain)
		}
		idx := slices.IndexFunc(ms, func(m sqlc.MembersRow) bool { return m.UserID == member })
		if idx < 0 {
			return apierror.New(apierror.NotFound)
		}
		if len(ms) == 1 {
			return s.disband(ctx, tx, q, id, now)
		}
		if err := q.EndMembership(ctx, sqlc.EndMembershipParams{TeamID: id, UserID: member, Now: now, Reason: reason}); err != nil {
			return err
		}
		if err := s.d.Events.PublishTx(ctx, tx, EventMemberLeft, now, MemberEvent{TeamID: id, UserID: member, Reason: reason}); err != nil {
			return err
		}
		if ms[idx].Role != "captain" {
			return nil
		}
		next := ms[0].UserID // members are ordered by joining time
		if next == member {
			next = ms[1].UserID
		}
		return s.setCaptain(ctx, tx, q, id, member, next, now)
	})
}

func (s *Service) setCaptain(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, id, from, to uuid.UUID, now time.Time) error {
	if err := q.SetRole(ctx, sqlc.SetRoleParams{TeamID: id, UserID: from, Role: "member"}); err != nil {
		return err
	}
	if err := q.SetRole(ctx, sqlc.SetRoleParams{TeamID: id, UserID: to, Role: "captain"}); err != nil {
		return err
	}
	if err := q.SetCaptain(ctx, sqlc.SetCaptainParams{ID: id, CaptainUserID: to, Now: now}); err != nil {
		return err
	}
	return s.d.Events.PublishTx(ctx, tx, EventCaptainChanged, now, MemberEvent{TeamID: id, UserID: to})
}

// TransferCaptain hands the captaincy to another member (captain).
func (s *Service) TransferCaptain(ctx context.Context, captain, id, member uuid.UUID) error {
	now := s.now()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		ms, role, err := locked(ctx, q, id, captain)
		if err != nil {
			return err
		}
		if role != "captain" {
			return apierror.New(apierror.NotCaptain)
		}
		if member == captain {
			return nil
		}
		if !slices.ContainsFunc(ms, func(m sqlc.MembersRow) bool { return m.UserID == member }) {
			return apierror.New(apierror.NotFound)
		}
		return s.setCaptain(ctx, tx, q, id, captain, member, now)
	})
}

// Disband ends a team (captain): memberships end, pending invites are
// cancelled; past results keep the team id (game.match_sides).
func (s *Service) Disband(ctx context.Context, captain, id uuid.UUID) error {
	now := s.now()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		_, role, err := locked(ctx, q, id, captain)
		if err != nil {
			return err
		}
		if role != "captain" {
			return apierror.New(apierror.NotCaptain)
		}
		return s.disband(ctx, tx, q, id, now)
	})
}

func (s *Service) disband(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, id uuid.UUID, now time.Time) error {
	if err := q.EndAllMemberships(ctx, sqlc.EndAllMembershipsParams{TeamID: id, Now: now}); err != nil {
		return err
	}
	if err := q.CancelInvites(ctx, id); err != nil {
		return err
	}
	if err := q.DisbandTeam(ctx, sqlc.DisbandTeamParams{ID: id, Now: now}); err != nil {
		return err
	}
	return s.d.Events.PublishTx(ctx, tx, EventTeamDisbanded, now, TeamEvent{TeamID: id})
}
