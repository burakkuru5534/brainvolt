package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
)

// Badge is a catalogue badge with the caller's earn date.
type Badge struct {
	Code        string     `json:"code"`
	Family      string     `json:"family"`
	Tier        int        `json:"tier"`
	IconKey     string     `json:"icon_key"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	EarnedAt    *time.Time `json:"earned_at"`
}

// EarnedBadge is a badge in match awards.
type EarnedBadge struct {
	Code    string `json:"code"`
	Tier    int    `json:"tier"`
	IconKey string `json:"icon_key"`
	Name    string `json:"name"`
}

// Badges returns the catalogue in a locale with the user's earn dates;
// hidden badges appear only once earned. earnedOnly keeps earned ones.
func (s *Service) Badges(ctx context.Context, user uuid.UUID, locale string, earnedOnly bool) ([]Badge, error) {
	rows, err := sqlc.New(s.d.DB).BadgeCatalog(ctx, sqlc.BadgeCatalogParams{Locale: locale, UserID: user})
	if err != nil {
		return nil, err
	}
	out := []Badge{}
	for _, r := range rows {
		if (earnedOnly || r.IsHidden) && r.EarnedAt == nil {
			continue
		}
		out = append(out, Badge{Code: r.Code, Family: r.Family, Tier: int(r.Tier), IconKey: r.IconKey, Name: r.Name,
			Description: r.Description, EarnedAt: r.EarnedAt})
	}
	return out, nil
}

// badge is an active badge with its parsed criteria.
type badge struct {
	id       uuid.UUID
	code     string
	tier     int
	criteria domain.Criteria
}

func activeBadges(ctx context.Context, q *sqlc.Queries) ([]badge, error) {
	rows, err := q.ActiveBadges(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]badge, 0, len(rows))
	for _, r := range rows {
		b := badge{id: r.ID, code: r.Code, tier: int(r.Tier)}
		if err := json.Unmarshal(r.Criteria, &b.criteria); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// award gives a badge once, with its XP under "<ref>:<code>"; it reports
// whether the badge is new.
func award(ctx context.Context, q *sqlc.Queries, user uuid.UUID, b *badge, ref string, now time.Time) (bool, int, error) {
	n, err := q.AwardBadge(ctx, sqlc.AwardBadgeParams{UserID: user, BadgeID: b.id, Now: now, SourceRef: &ref})
	if err != nil || n == 0 {
		return false, 0, err
	}
	added, _, _, err := grant(ctx, q, user, domain.BadgeXP(b.tier), sourceBadge, ref+":"+b.code, now)
	return true, added, err
}
