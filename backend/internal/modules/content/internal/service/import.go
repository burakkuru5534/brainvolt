package service

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Name is a localized name and optional description.
type Name struct {
	Name        string
	Description string
}

// SubcategorySpec is one sub-category of the taxonomy file.
type SubcategorySpec struct {
	Slug  string
	Names map[string]Name
}

// CategorySpec is one category of the taxonomy file; its position in the
// list is its sort order.
type CategorySpec struct {
	Slug          string
	IconKey       string
	Names         map[string]Name
	Subcategories []SubcategorySpec
}

// SourceSpec is one entry of the sources file (D-006).
type SourceSpec struct {
	ID              uuid.UUID
	Kind            string
	Name            string
	URL             string
	LicenseCode     string
	LicenseURL      string
	AttributionText string
	ShareAlike      bool
	Notes           string
}

// UpsertTaxonomy creates or updates categories and sub-categories by slug.
// Entries missing from the file are left as they are (deactivation is a
// separate, deliberate step).
func (s *Service) UpsertTaxonomy(ctx context.Context, cats []CategorySpec) error {
	now := s.now()
	return postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		for i, c := range cats {
			if err := requireNames(c.Slug, c.Names); err != nil {
				return err
			}
			catID, err := q.UpsertCategory(ctx, sqlc.UpsertCategoryParams{
				ID: ids.New(), Slug: c.Slug, IconKey: c.IconKey, SortOrder: int32(i), IsActive: true, Now: now,
			})
			if err != nil {
				return fmt.Errorf("category %s: %w", c.Slug, err)
			}
			for _, l := range domain.Locales {
				n := c.Names[l]
				if err := q.UpsertCategoryTranslation(ctx, sqlc.UpsertCategoryTranslationParams{
					CategoryID: catID, Locale: l, Name: n.Name, Description: optional(n.Description),
				}); err != nil {
					return fmt.Errorf("category %s/%s: %w", c.Slug, l, err)
				}
			}
			for j, sc := range c.Subcategories {
				if err := requireNames(c.Slug+"/"+sc.Slug, sc.Names); err != nil {
					return err
				}
				subID, err := q.UpsertSubcategory(ctx, sqlc.UpsertSubcategoryParams{
					ID: ids.New(), CategoryID: catID, Slug: sc.Slug, SortOrder: int32(j), IsActive: true, Now: now,
				})
				if err != nil {
					return fmt.Errorf("subcategory %s/%s: %w", c.Slug, sc.Slug, err)
				}
				for _, l := range domain.Locales {
					n := sc.Names[l]
					if err := q.UpsertSubcategoryTranslation(ctx, sqlc.UpsertSubcategoryTranslationParams{
						SubcategoryID: subID, Locale: l, Name: n.Name, Description: optional(n.Description),
					}); err != nil {
						return fmt.Errorf("subcategory %s/%s/%s: %w", c.Slug, sc.Slug, l, err)
					}
				}
			}
		}
		return nil
	})
}

// requireNames: the catalogue is shown in every locale (D-002).
func requireNames(what string, names map[string]Name) error {
	for _, l := range domain.Locales {
		if names[l].Name == "" {
			return fmt.Errorf("%s: missing %s name", what, l)
		}
	}
	return nil
}

// UpsertSources creates or updates sources by id.
func (s *Service) UpsertSources(ctx context.Context, sources []SourceSpec) error {
	now := s.now()
	return postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		for i := range sources {
			src := &sources[i]
			if src.LicenseCode == "" || src.AttributionText == "" {
				return fmt.Errorf("source %s: license and attribution are required (D-006)", src.Name)
			}
			if err := q.UpsertSource(ctx, sqlc.UpsertSourceParams{
				ID: src.ID, Kind: src.Kind, Name: src.Name, Url: optional(src.URL),
				LicenseCode: src.LicenseCode, LicenseUrl: optional(src.LicenseURL),
				AttributionText: src.AttributionText, ShareAlike: src.ShareAlike,
				Notes: optional(src.Notes), Now: now,
			}); err != nil {
				return fmt.Errorf("source %s: %w", src.Name, err)
			}
		}
		return nil
	})
}

// SubcategoryID resolves "category/subcategory" slugs.
func (s *Service) SubcategoryID(ctx context.Context, category, subcategory string) (uuid.UUID, error) {
	id, err := sqlc.New(s.db).SubcategoryIDBySlugs(ctx, sqlc.SubcategoryIDBySlugsParams{
		CategorySlug: category, SubcategorySlug: subcategory,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("unknown sub-category %s/%s", category, subcategory)
	}
	return id, err
}

// ImportResult says what Import did with one question.
type ImportResult struct {
	ID      uuid.UUID
	Created bool
}

// Import stores an imported question in review (schema-content §6); a
// question whose fingerprint already exists is skipped, so re-running an
// import is harmless.
func (s *Service) Import(ctx context.Context, d domain.Draft) (ImportResult, error) {
	id, err := s.Create(ctx, d, CreateOptions{Status: domain.StatusInReview})
	if e := apierror.From(err); e != nil && e.Code == apierror.QuestionDuplicate {
		existing, _ := e.Details["question_id"].(uuid.UUID)
		return ImportResult{ID: existing}, nil
	}
	return ImportResult{ID: id, Created: err == nil}, err
}

// ApproveAll verifies a question and all its translations at once. Only
// cmd/seed's development switch uses it, so local matches have content
// before a moderator has reviewed anything; it is refused outside dev.
func (s *Service) ApproveAll(ctx context.Context, actor, id uuid.UUID) error {
	q, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if q.Status == domain.StatusDraft {
		if err := s.Submit(ctx, actor, id); err != nil {
			return err
		}
	}
	if q.Status != domain.StatusVerified {
		if err := s.Review(ctx, actor, id, Review{Decision: domain.DecisionApprove}); err != nil {
			return err
		}
	}
	for _, l := range slices.Sorted(maps.Keys(q.Translations)) {
		if l == q.OriginalLocale || q.Translations[l].Status == domain.StatusVerified {
			continue
		}
		if err := s.Review(ctx, actor, id, Review{Decision: domain.DecisionApprove, Locale: &l}); err != nil {
			return err
		}
	}
	return nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
