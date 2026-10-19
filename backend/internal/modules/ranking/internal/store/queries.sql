-- ---------- Consumption ----------

-- name: MarkProcessed :execrows
INSERT INTO ranking.processed_matches (match_id, processed_at) VALUES (@match_id, @now) ON CONFLICT DO NOTHING;

-- ---------- Seasons ----------

-- name: ActiveSeason :one
SELECT * FROM ranking.seasons WHERE status = 'active';

-- name: LastSeasonNumber :one
SELECT coalesce(max(number), 0)::int FROM ranking.seasons;

-- name: InsertSeason :exec
INSERT INTO ranking.seasons (id, number, starts_at, ends_at, status, reset_policy)
VALUES (@id, @number, @starts_at, @ends_at, 'active', @reset_policy);

-- name: InsertSeasonName :exec
INSERT INTO ranking.season_translations (season_id, locale, name) VALUES (@season_id, @locale, @name);

-- name: GetSeason :one
SELECT * FROM ranking.seasons WHERE id = @id;

-- name: SeasonNames :many
SELECT locale, name FROM ranking.season_translations WHERE season_id = @season_id;

-- name: SetSeasonStatus :exec
UPDATE ranking.seasons SET status = @status, closed_at = sqlc.narg(closed_at) WHERE id = @id;

-- ---------- Ratings ----------

-- name: GetRating :one
SELECT * FROM ranking.ratings WHERE entity_kind = @entity_kind AND entity_id = @entity_id AND bucket = @bucket;

-- name: UpsertRating :exec
INSERT INTO ranking.ratings (entity_kind, entity_id, bucket, rating, rd, volatility, rated_matches, last_rated_at, updated_at)
VALUES (@entity_kind, @entity_id, @bucket, @rating, @rd, @volatility, 1, @now, @now)
ON CONFLICT (entity_kind, entity_id, bucket) DO UPDATE
SET rating = EXCLUDED.rating, rd = EXCLUDED.rd, volatility = EXCLUDED.volatility,
    rated_matches = ranking.ratings.rated_matches + 1, last_rated_at = EXCLUDED.last_rated_at, updated_at = EXCLUDED.updated_at;

-- name: InsertRatingChange :exec
INSERT INTO ranking.rating_changes (match_id, entity_kind, entity_id, bucket, season_id, rating_before, rating_after,
  rd_before, rd_after, vol_before, vol_after, created_at)
VALUES (@match_id, @entity_kind, @entity_id, @bucket, @season_id, @rating_before, @rating_after,
  @rd_before, @rd_after, @vol_before, @vol_after, @now);

-- Season reset: RD raised to at least @min_rd (progression-formulas §9).
-- name: RaiseRD :many
UPDATE ranking.ratings SET rd = @min_rd::numeric, updated_at = @now
WHERE rd < @min_rd::numeric
RETURNING entity_kind, entity_id, bucket, rating, volatility;

-- ---------- Standings ----------

-- name: GetStanding :one
SELECT * FROM ranking.season_standings WHERE season_id = @season_id AND entity_kind = @entity_kind AND entity_id = @entity_id;

-- name: UpsertStanding :exec
INSERT INTO ranking.season_standings (season_id, entity_kind, entity_id, season_points, league_code, division,
  peak_league, peak_division, matches, wins, losses, draws, best_win_streak, current_win_streak, updated_at)
VALUES (@season_id, @entity_kind, @entity_id, @season_points, @league_code, @division,
  @peak_league, @peak_division, @matches, @wins, @losses, @draws, @best_win_streak, @current_win_streak, @now)
ON CONFLICT (season_id, entity_kind, entity_id) DO UPDATE
SET season_points = EXCLUDED.season_points, league_code = EXCLUDED.league_code, division = EXCLUDED.division,
    peak_league = EXCLUDED.peak_league, peak_division = EXCLUDED.peak_division, matches = EXCLUDED.matches,
    wins = EXCLUDED.wins, losses = EXCLUDED.losses, draws = EXCLUDED.draws, best_win_streak = EXCLUDED.best_win_streak,
    current_win_streak = EXCLUDED.current_win_streak, updated_at = EXCLUDED.updated_at;

-- name: SeasonStandings :many
SELECT * FROM ranking.season_standings WHERE season_id = @season_id AND entity_kind = @entity_kind
ORDER BY season_points DESC, updated_at
LIMIT @lim;

-- name: SetFinalRank :exec
UPDATE ranking.season_standings SET final_rank_global = @rank
WHERE season_id = @season_id AND entity_kind = @entity_kind AND entity_id = @entity_id;

-- ---------- Ledger ----------

-- name: InsertLedger :execrows
INSERT INTO ranking.points_ledger (id, match_id, entity_kind, entity_id, season_id, category_id, points, country_code, public, occurred_at)
VALUES (@id, @match_id, @entity_kind, @entity_id, @season_id, sqlc.narg(category_id), @points, sqlc.narg(country_code), @public, @occurred_at)
ON CONFLICT DO NOTHING;

-- Rows of a period (board rebuild and tier-change replay).
-- name: LedgerSince :many
SELECT * FROM ranking.points_ledger WHERE entity_kind = @entity_kind AND entity_id = @entity_id AND occurred_at >= @since;

-- ---------- Leagues ----------

-- name: Leagues :many
SELECT l.code, l.sort_order, l.divisions, l.rules, t.name
FROM ranking.leagues l JOIN ranking.league_translations t ON t.code = l.code AND t.locale = @locale
ORDER BY l.sort_order;

-- ---------- Snapshots ----------

-- name: InsertSnapshot :exec
INSERT INTO ranking.leaderboard_snapshots (board_key, period_start, period_end, rank, entity_kind, entity_id, score)
VALUES (@board_key, @period_start, @period_end, @rank, @entity_kind, @entity_id, @score)
ON CONFLICT DO NOTHING;

-- A user's rows hidden while restricted, made public on upgrade (D-016, D-017).
-- name: PublishUserLedger :many
UPDATE ranking.points_ledger SET public = true
WHERE entity_kind = 'user' AND entity_id = @user_id AND NOT public
RETURNING category_id, points, country_code, occurred_at;
