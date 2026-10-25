-- tournament schema queries (schema-platform §3, D-090). Only the tournament schema.

-- name: InsertTournament :exec
INSERT INTO tournament.tournaments (id, status, bracket_type, entity_kind, match_config, min_entrants, max_entrants,
  registration_opens_at, registration_closes_at, starts_at, round_ready_window_s, rewards, created_by, created_at, kind, fee)
VALUES (@id, @status, 'single_elimination', @entity_kind, @match_config, @min_entrants, @max_entrants,
  @registration_opens_at, @registration_closes_at, @starts_at, @round_ready_window_s, @rewards, @created_by, @now, @kind, @fee);

-- name: InsertTranslation :exec
INSERT INTO tournament.tournament_translations (tournament_id, locale, name, description)
VALUES (@tournament_id, @locale, @name, sqlc.narg(description));

-- name: GetTournament :one
SELECT * FROM tournament.tournaments WHERE id = @id;

-- name: LockTournament :one
SELECT * FROM tournament.tournaments WHERE id = @id FOR UPDATE;

-- name: Translations :many
SELECT * FROM tournament.tournament_translations WHERE tournament_id = ANY(@ids::uuid[]);

-- Newest start first; drafts only for staff.
-- name: ListTournaments :many
SELECT t.*, (SELECT count(*) FROM tournament.entries e
  WHERE e.tournament_id = t.id AND e.status NOT IN ('withdrawn','disqualified'))::int AS entrants
FROM tournament.tournaments t
WHERE (sqlc.narg(status)::text IS NULL OR t.status = sqlc.narg(status)::text)
  AND (@with_drafts::boolean OR t.status <> 'draft')
  -- Friend tournaments only for the owner and the owner's friends (D-109).
  AND (t.kind = 'official' OR t.created_by = ANY(@owners::uuid[]))
ORDER BY t.starts_at DESC
LIMIT 50;

-- name: SetTournamentStatus :exec
UPDATE tournament.tournaments SET status = @status,
  finished_at = CASE WHEN @status IN ('finished','cancelled') THEN @now::timestamptz ELSE finished_at END
WHERE id = @id;

-- Registration closed and start time reached.
-- name: DueToStart :many
SELECT id FROM tournament.tournaments WHERE status = 'registration' AND starts_at <= @now;

-- name: InsertEntry :exec
INSERT INTO tournament.entries (id, tournament_id, entity_kind, entity_id, lineup, registered_at)
VALUES (@id, @tournament_id, @entity_kind, @entity_id, @lineup, @now);

-- name: Entries :many
SELECT * FROM tournament.entries WHERE tournament_id = @tournament_id ORDER BY seed NULLS LAST, registered_at;

-- name: CountEntries :one
SELECT count(*)::int FROM tournament.entries WHERE tournament_id = @tournament_id AND status NOT IN ('withdrawn','disqualified');

-- The caller's live entry: they are in its lineup.
-- name: EntryOf :one
SELECT * FROM tournament.entries
WHERE tournament_id = @tournament_id AND @user_id::uuid = ANY(lineup) AND status NOT IN ('withdrawn','disqualified');

-- Live entries of the caller across tournaments (the list's my_entry).
-- name: EntriesOfUser :many
SELECT * FROM tournament.entries
WHERE tournament_id = ANY(@ids::uuid[]) AND @user_id::uuid = ANY(lineup) AND status NOT IN ('withdrawn','disqualified');

-- name: LineupTaken :one
SELECT EXISTS (SELECT 1 FROM tournament.entries
  WHERE tournament_id = @tournament_id AND lineup && @lineup::uuid[] AND status NOT IN ('withdrawn','disqualified'));

-- name: EntityEntry :one
SELECT * FROM tournament.entries WHERE tournament_id = @tournament_id AND entity_kind = @entity_kind AND entity_id = @entity_id;

-- A withdrawn entry registering again.
-- name: ReviveEntry :exec
UPDATE tournament.entries SET status = 'registered', lineup = @lineup, registered_at = @now, seed = NULL, final_place = NULL
WHERE id = @id;

-- name: SetEntryStatus :exec
UPDATE tournament.entries SET status = @status WHERE id = @id;

-- name: SeedEntry :exec
UPDATE tournament.entries SET seed = @seed, status = 'active' WHERE id = @id;

-- name: PlaceEntry :exec
UPDATE tournament.entries SET status = @status, final_place = @final_place WHERE id = @id;

-- Entries that did not check in drop out at the start.
-- name: WithdrawUnchecked :exec
UPDATE tournament.entries SET status = 'withdrawn' WHERE tournament_id = @tournament_id AND status = 'registered';

-- name: InsertSlot :exec
INSERT INTO tournament.bracket_matches (tournament_id, round_no, slot_no, entry_a_id, entry_b_id, status)
VALUES (@tournament_id, @round_no, @slot_no, sqlc.narg(entry_a_id), sqlc.narg(entry_b_id), 'pending');

-- name: Slots :many
SELECT * FROM tournament.bracket_matches WHERE tournament_id = @tournament_id ORDER BY round_no, slot_no;

-- name: LockSlot :one
SELECT * FROM tournament.bracket_matches
WHERE tournament_id = @tournament_id AND bracket_side = 'main' AND round_no = @round_no AND slot_no = @slot_no FOR UPDATE;

-- The slot an entry plays now (ready or in progress).
-- name: CurrentSlotOf :one
SELECT * FROM tournament.bracket_matches
WHERE tournament_id = @tournament_id AND (entry_a_id = @entry_id OR entry_b_id = @entry_id) AND status IN ('ready','in_progress')
FOR UPDATE;

-- name: SlotOfMatch :one
SELECT * FROM tournament.bracket_matches WHERE match_id = @match_id FOR UPDATE;

-- name: SetSlotEntry :exec
UPDATE tournament.bracket_matches
SET entry_a_id = CASE WHEN @side_a::boolean THEN @entry_id::uuid ELSE entry_a_id END,
    entry_b_id = CASE WHEN @side_a::boolean THEN entry_b_id ELSE @entry_id::uuid END
WHERE tournament_id = @tournament_id AND bracket_side = 'main' AND round_no = @round_no AND slot_no = @slot_no;

-- name: OpenSlot :exec
UPDATE tournament.bracket_matches SET status = 'ready', scheduled_at = @now, ready_until = @ready_until, present = '{}', match_id = NULL
WHERE tournament_id = @tournament_id AND bracket_side = 'main' AND round_no = @round_no AND slot_no = @slot_no;

-- name: SetPresent :exec
UPDATE tournament.bracket_matches SET present = @present
WHERE tournament_id = @tournament_id AND bracket_side = 'main' AND round_no = @round_no AND slot_no = @slot_no;

-- name: StartSlot :exec
UPDATE tournament.bracket_matches SET status = 'in_progress', match_id = @match_id
WHERE tournament_id = @tournament_id AND bracket_side = 'main' AND round_no = @round_no AND slot_no = @slot_no;

-- name: FinishSlot :exec
UPDATE tournament.bracket_matches SET status = @status, winner_entry_id = @winner_entry_id
WHERE tournament_id = @tournament_id AND bracket_side = 'main' AND round_no = @round_no AND slot_no = @slot_no;

-- Ready windows that closed without a match.
-- name: ExpiredSlots :many
SELECT tournament_id, round_no, slot_no FROM tournament.bracket_matches WHERE status = 'ready' AND ready_until <= @now;

-- ---------- Friend tournaments (D-109) ----------

-- name: FriendTrophies :one
-- Friend tournaments a user won.
SELECT count(*)::int FROM tournament.entries e
JOIN tournament.tournaments t ON t.id = e.tournament_id
WHERE t.kind = 'friends' AND t.status = 'finished' AND e.final_place = 1 AND @user_id::uuid = ANY(e.lineup);

-- name: OpenFriendTournaments :one
-- An owner's friend tournaments not yet finished (a limit on setting up).
SELECT count(*)::int FROM tournament.tournaments
WHERE kind = 'friends' AND created_by = @user_id AND status IN ('registration','seeding','running');
