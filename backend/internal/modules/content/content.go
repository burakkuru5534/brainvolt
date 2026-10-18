// Package content is the public API of the content module
// (backend-modules.md §1): categories and sub-categories, questions with
// TR/EN text, sources and licenses (D-006), verification (D-007), the
// moderator workflow and the import used by cmd/seed. Question selection
// for matches is added with the game engine (Phase 3).
package content

import (
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

// Register adds content's routes; ident authenticates them.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.Service, ident, idem).Register(s)
}
