// Package content is the public API of the content module
// (backend-modules.md §1): categories and sub-categories, questions with
// TR/EN text, sources and licenses (D-006), verification (D-007), the
// moderator workflow and the import used by cmd/seed. Question selection
// for matches is added with the game engine (Phase 3).
package content

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Draft is a question as created or imported.
type Draft = domain.Draft

// Text is a question's wording in one locale.
type Text = domain.Text

// QuestionType is multiple_choice or true_false.
type QuestionType = domain.QuestionType

// Origin says how a translation was written.
type Origin = domain.Origin

// CategorySpec is one category of content/taxonomy.yaml.
type CategorySpec = service.CategorySpec

// SubcategorySpec is one sub-category of content/taxonomy.yaml.
type SubcategorySpec = service.SubcategorySpec

// Name is a localized name and description.
type Name = service.Name

// SourceSpec is one entry of content/sources.yaml (D-006).
type SourceSpec = service.SourceSpec

// ImportResult says whether Import created or skipped a question.
type ImportResult = service.ImportResult

// QuestionReviewed is the payload of the review events.
type QuestionReviewed = service.QuestionReviewed

// SelectRequest asks for the questions of one match.
type SelectRequest = service.SelectRequest

// SelectRules are the mix rules of a selection.
type SelectRules = domain.SelectRules

// SelectedQuestion is one selected question.
type SelectedQuestion = service.SelectedQuestion

// AnswerStat is one (question, locale) statistics line of a match.
type AnswerStat = service.AnswerStat

// Constants.
const (
	MultipleChoice = domain.MultipleChoice
	TrueFalse      = domain.TrueFalse

	OriginOriginal           = domain.OriginOriginal
	OriginHumanTranslation   = domain.OriginHumanTranslation
	OriginMachineTranslation = domain.OriginMachineTranslation

	EventQuestionVerified = service.EventQuestionVerified
	EventQuestionRejected = service.EventQuestionRejected
)

// Locales are the content languages (D-001, D-002).
var Locales = domain.Locales

// Fingerprint identifies a question by its original wording and option set
// (duplicate detection).
func Fingerprint(locale string, t Text) string { return domain.Fingerprint(locale, t) }

// Module is the content module.
type Module struct {
	*service.Service
}

// New builds the module. The embedded service exposes the import API
// (UpsertTaxonomy, UpsertSources, SubcategoryID, Import, ApproveAll) to
// cmd/seed.
func New(db *pgxpool.Pool, c clock.Clock, bus *events.Bus) *Module {
	return &Module{Service: service.New(db, c, bus)}
}

// MatchStats is the part of game.MatchCompleted that content reads
// (content does not import game: layer 2 below layer 3).
type MatchStats struct {
	MatchID       uuid.UUID    `json:"match_id"`
	QuestionStats []AnswerStat `json:"question_stats"`
}

// SubscriberMatchStats is content's subscription to game.MatchCompleted.
const SubscriberMatchStats = "content.on_match_completed"

// OnMatchCompleted records a completed match's answer statistics.
func (m *Module) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	var p MatchStats
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return fmt.Errorf("content: decoding %s: %w", e.Name, err)
	}
	return m.RecordStats(ctx, p.MatchID, p.QuestionStats, e.OccurredAt)
}

// Register adds content's routes; ident authenticates them.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.Service, ident, idem).Register(s)
}
