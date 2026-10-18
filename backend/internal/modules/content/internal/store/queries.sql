-- content queries (docs/domain/schema-content.md). Only the content schema.

-- ---------- Taxonomy ----------

-- name: ListCategories :many
SELECT c.id, c.slug, c.icon_key, c.sort_order, t.name, t.description
FROM content.categories c
JOIN content.category_translations t ON t.category_id = c.id AND t.locale = @locale
WHERE c.is_active
ORDER BY c.sort_order, c.slug;

-- name: ListSubcategories :many
SELECT s.id, s.category_id, s.slug, s.sort_order, t.name, t.description
FROM content.subcategories s
JOIN content.categories c ON c.id = s.category_id AND c.is_active
JOIN content.subcategory_translations t ON t.subcategory_id = s.id AND t.locale = @locale
WHERE s.is_active
ORDER BY s.sort_order, s.slug;

-- name: SubcategoryExists :one
SELECT EXISTS (SELECT 1 FROM content.subcategories WHERE id = $1);

-- name: UpsertCategory :one
INSERT INTO content.categories (id, slug, icon_key, sort_order, is_active, created_at, updated_at)
VALUES (@id, @slug, @icon_key, @sort_order, @is_active, @now, @now)
ON CONFLICT (slug) DO UPDATE
SET icon_key = EXCLUDED.icon_key, sort_order = EXCLUDED.sort_order,
    is_active = EXCLUDED.is_active, updated_at = EXCLUDED.updated_at
RETURNING id;

-- name: UpsertCategoryTranslation :exec
INSERT INTO content.category_translations (category_id, locale, name, description)
VALUES ($1, $2, $3, $4)
ON CONFLICT (category_id, locale) DO UPDATE
SET name = EXCLUDED.name, description = EXCLUDED.description;

-- name: UpsertSubcategory :one
INSERT INTO content.subcategories (id, category_id, slug, sort_order, is_active, created_at, updated_at)
VALUES (@id, @category_id, @slug, @sort_order, @is_active, @now, @now)
ON CONFLICT (category_id, slug) DO UPDATE
SET sort_order = EXCLUDED.sort_order, is_active = EXCLUDED.is_active, updated_at = EXCLUDED.updated_at
RETURNING id;

-- name: UpsertSubcategoryTranslation :exec
INSERT INTO content.subcategory_translations (subcategory_id, locale, name, description)
VALUES ($1, $2, $3, $4)
ON CONFLICT (subcategory_id, locale) DO UPDATE
SET name = EXCLUDED.name, description = EXCLUDED.description;

-- name: SubcategoryIDBySlugs :one
SELECT s.id FROM content.subcategories s
JOIN content.categories c ON c.id = s.category_id
WHERE c.slug = @category_slug AND s.slug = @subcategory_slug;

-- ---------- Eligibility (§3) ----------

-- Questions a match may use in @locale: verified, not retired, general
-- pool, with a verified translation in that locale. Ranked additionally
-- excludes true/false (D-024).
-- name: CountEligible :many
SELECT q.subcategory_id,
       count(*) FILTER (WHERE q.question_type = 'multiple_choice')::bigint AS ranked,
       count(*)::bigint AS casual
FROM content.questions q
JOIN content.question_translations t
  ON t.question_id = q.id AND t.locale = @locale AND t.verification_status = 'verified'
WHERE q.verification_status = 'verified' AND q.retired_at IS NULL AND q.pool = 'general'
GROUP BY q.subcategory_id;

-- The newest change to content that /v1/categories shows (ETag input).
-- name: ContentVersion :one
SELECT greatest(
  (SELECT coalesce(max(updated_at), 'epoch') FROM content.categories),
  (SELECT coalesce(max(updated_at), 'epoch') FROM content.subcategories),
  (SELECT coalesce(max(updated_at), 'epoch') FROM content.questions),
  (SELECT coalesce(max(updated_at), 'epoch') FROM content.question_translations)
)::timestamptz AS version,
(SELECT count(*) FROM content.questions)::bigint AS questions;

-- ---------- Sources (D-006) ----------

-- name: ListSources :many
SELECT * FROM content.sources ORDER BY created_at, name;

-- name: SourceExists :one
SELECT EXISTS (SELECT 1 FROM content.sources WHERE id = $1);

-- name: UpsertSource :exec
INSERT INTO content.sources (id, kind, name, url, license_code, license_url, attribution_text, share_alike, notes, created_at)
VALUES (@id, @kind, @name, @url, @license_code, @license_url, @attribution_text, @share_alike, @notes, @now)
ON CONFLICT (id) DO UPDATE
SET kind = EXCLUDED.kind, name = EXCLUDED.name, url = EXCLUDED.url,
    license_code = EXCLUDED.license_code, license_url = EXCLUDED.license_url,
    attribution_text = EXCLUDED.attribution_text, share_alike = EXCLUDED.share_alike,
    notes = EXCLUDED.notes;

-- ---------- Questions ----------

-- name: InsertQuestion :exec
INSERT INTO content.questions (id, subcategory_id, question_type, difficulty, original_locale,
  source_id, source_ref, author_user_id, fingerprint, verification_status, created_at, updated_at)
VALUES (@id, @subcategory_id, @question_type, @difficulty, @original_locale,
  @source_id, @source_ref, @author_user_id, @fingerprint, @verification_status, @now, @now);

-- name: QuestionIDByFingerprint :one
SELECT id FROM content.questions WHERE fingerprint = $1;

-- name: GetQuestion :one
SELECT * FROM content.questions WHERE id = $1;

-- name: GetQuestionForUpdate :one
SELECT * FROM content.questions WHERE id = $1 FOR UPDATE;

-- name: UpdateQuestion :exec
UPDATE content.questions
SET subcategory_id = @subcategory_id, difficulty = @difficulty, fingerprint = @fingerprint,
    verification_status = @verification_status, verified_by = @verified_by, verified_at = @verified_at,
    rejection_reason = @rejection_reason, revision = @revision, updated_at = @now
WHERE id = @id;

-- name: SetQuestionStatus :exec
UPDATE content.questions
SET verification_status = @verification_status, verified_by = @verified_by, verified_at = @verified_at,
    rejection_reason = @rejection_reason, updated_at = @now
WHERE id = @id;

-- name: RetireQuestion :exec
UPDATE content.questions SET retired_at = @now, updated_at = @now WHERE id = @id AND retired_at IS NULL;

-- A question counts as used once a match result reached its statistics
-- (schema-content §4.3, §5).
-- name: QuestionUsed :one
SELECT EXISTS (SELECT 1 FROM content.question_stats WHERE question_id = $1 AND times_asked > 0);

-- name: ListQuestions :many
SELECT q.*, coalesce(t.prompt, o.prompt)::text AS prompt
FROM content.questions q
JOIN content.question_translations o ON o.question_id = q.id AND o.locale = q.original_locale
LEFT JOIN content.question_translations t ON t.question_id = q.id AND t.locale = sqlc.narg(locale)::text
WHERE (sqlc.narg(status)::text IS NULL OR q.verification_status = sqlc.narg(status)::text)
  AND (sqlc.narg(subcategory_id)::uuid IS NULL OR q.subcategory_id = sqlc.narg(subcategory_id)::uuid)
  AND (sqlc.narg(locale)::text IS NULL OR t.question_id IS NOT NULL)
  AND (sqlc.narg(translation_status)::text IS NULL OR t.verification_status = sqlc.narg(translation_status)::text)
  AND (sqlc.arg(include_retired)::boolean OR q.retired_at IS NULL)
  AND (sqlc.narg(cursor)::uuid IS NULL OR q.id < sqlc.narg(cursor)::uuid)
ORDER BY q.id DESC
LIMIT sqlc.arg(row_limit);

-- ---------- Translations & options ----------

-- name: UpsertQuestionTranslation :exec
INSERT INTO content.question_translations (question_id, locale, prompt, explanation, origin,
  verification_status, verified_by, verified_at, rejection_reason, created_at, updated_at)
VALUES (@question_id, @locale, @prompt, @explanation, @origin,
  @verification_status, @verified_by, @verified_at, @rejection_reason, @now, @now)
ON CONFLICT (question_id, locale) DO UPDATE
SET prompt = EXCLUDED.prompt, explanation = EXCLUDED.explanation, origin = EXCLUDED.origin,
    verification_status = EXCLUDED.verification_status, verified_by = EXCLUDED.verified_by,
    verified_at = EXCLUDED.verified_at, rejection_reason = EXCLUDED.rejection_reason,
    updated_at = EXCLUDED.updated_at;

-- name: ListQuestionTranslations :many
SELECT * FROM content.question_translations WHERE question_id = $1 ORDER BY locale;

-- name: ListQuestionTranslationStatuses :many
SELECT question_id, locale, verification_status FROM content.question_translations
WHERE question_id = ANY(@ids::uuid[]) ORDER BY question_id, locale;

-- name: SetTranslationStatus :exec
UPDATE content.question_translations
SET verification_status = @verification_status, verified_by = @verified_by, verified_at = @verified_at,
    rejection_reason = @rejection_reason, updated_at = @now
WHERE question_id = @question_id AND locale = @locale;

-- name: InsertOption :exec
INSERT INTO content.options (id, question_id, position, is_correct) VALUES ($1, $2, $3, $4);

-- name: SetOptionCorrect :exec
UPDATE content.options SET is_correct = (position = @correct_position) WHERE question_id = @question_id;

-- name: ClearOptionCorrect :exec
UPDATE content.options SET is_correct = false WHERE question_id = $1;

-- name: ListOptions :many
SELECT * FROM content.options WHERE question_id = $1 ORDER BY position;

-- name: UpsertOptionTranslation :exec
INSERT INTO content.option_translations (option_id, locale, text) VALUES ($1, $2, $3)
ON CONFLICT (option_id, locale) DO UPDATE SET text = EXCLUDED.text;

-- name: ListOptionTranslations :many
SELECT ot.* FROM content.option_translations ot
JOIN content.options o ON o.id = ot.option_id
WHERE o.question_id = $1
ORDER BY o.position, ot.locale;

-- ---------- History ----------

-- name: InsertRevision :exec
INSERT INTO content.question_revisions (question_id, revision, snapshot, created_by, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ListRevisions :many
SELECT question_id, revision, created_by, created_at FROM content.question_revisions
WHERE question_id = $1 ORDER BY revision;

-- name: InsertVerificationEvent :exec
INSERT INTO content.verification_events (id, question_id, locale, from_status, to_status, actor_id, reason, revision, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListVerificationEvents :many
SELECT * FROM content.verification_events WHERE question_id = $1 ORDER BY created_at, id;

-- name: ListQuestionStats :many
SELECT * FROM content.question_stats WHERE question_id = $1 ORDER BY locale;
