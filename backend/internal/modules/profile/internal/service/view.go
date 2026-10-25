package service

import (
	"context"

	"github.com/google/uuid"
)

// Relations of a viewer to a player (D-101).
const (
	RelationSelf     = "self"
	RelationFriend   = "friend"
	RelationSent     = "request_sent"
	RelationReceived = "request_received"
	RelationBlocked  = "blocked"
	RelationNone     = "none"
)

// RelationReader tells how a viewer relates to another user. social
// implements it; it is wired in internal/app because profile may not
// import social (backend-modules.md §2, profile → social*).
type RelationReader interface {
	Relation(ctx context.Context, viewer, other uuid.UUID) (string, error)
}

// StatsReader returns a player's public progress. progression implements
// it (profile → progression*).
type StatsReader interface {
	PlayerStats(ctx context.Context, user uuid.UUID) (PlayerStats, error)
}

// SetReaders wires the upward reads; without them a profile shows no
// relation ("none") and no stats.
func (s *Service) SetReaders(rel RelationReader, stats StatsReader) {
	s.relations, s.stats = rel, stats
}

// View is a player's profile as viewer sees it (D-101): the card, the
// viewer's relation and the stats unless the owner keeps the profile
// private and the viewer is not a friend. Blocked pairs never see stats.
func (s *Service) View(ctx context.Context, viewer, user uuid.UUID) (PublicProfile, error) {
	pub, err := s.Public(ctx, user)
	if err != nil {
		return PublicProfile{}, err
	}
	privacy, err := s.PrivacyOf(ctx, []uuid.UUID{user})
	if err != nil {
		return PublicProfile{}, err
	}
	pub.Private = privacy[user].ProfilePrivate
	w := s.worn(ctx, []uuid.UUID{user})[user]
	pub.Frame, pub.NameColor = w.Frame, w.NameColor
	pub.Relation = RelationNone
	switch {
	case viewer == user:
		pub.Relation = RelationSelf
	case s.relations != nil:
		if pub.Relation, err = s.relations.Relation(ctx, viewer, user); err != nil {
			return PublicProfile{}, err
		}
	}
	visible := pub.Relation == RelationSelf || pub.Relation == RelationFriend ||
		(!pub.Private && pub.Relation != RelationBlocked)
	if visible && s.stats != nil {
		st, err := s.stats.PlayerStats(ctx, user)
		if err != nil {
			return PublicProfile{}, err
		}
		pub.Stats = &st
	}
	return pub, nil
}
