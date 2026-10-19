-- ---------- Teams ----------

-- name: InsertTeam :exec
INSERT INTO team.teams (id, name, tag, emblem_id, captain_user_id, created_at, updated_at)
VALUES (@id, @name, @tag, @emblem_id, @captain_user_id, @now, @now);

-- name: GetTeam :one
SELECT * FROM team.teams WHERE id = @id;

-- name: GetTeamForUpdate :one
SELECT * FROM team.teams WHERE id = @id FOR UPDATE;

-- name: UpdateTeam :exec
UPDATE team.teams SET name = @name, tag = @tag, emblem_id = @emblem_id, updated_at = @now WHERE id = @id;

-- name: SetCaptain :exec
UPDATE team.teams SET captain_user_id = @captain_user_id, updated_at = @now WHERE id = @id;

-- name: DisbandTeam :exec
UPDATE team.teams SET status = 'disbanded', disbanded_at = @now::timestamptz, updated_at = @now::timestamptz WHERE id = @id;

-- name: NameOrTagTaken :one
SELECT
  EXISTS (SELECT 1 FROM team.teams WHERE status = 'active' AND name = @name::citext AND id <> @self::uuid) AS name_taken,
  EXISTS (SELECT 1 FROM team.teams WHERE status = 'active' AND tag = @tag::citext AND id <> @self::uuid) AS tag_taken;

-- ---------- Memberships ----------

-- name: InsertMembership :exec
INSERT INTO team.memberships (team_id, user_id, role, joined_at) VALUES (@team_id, @user_id, @role, @now);

-- name: Members :many
SELECT user_id, role, joined_at FROM team.memberships
WHERE team_id = @team_id AND left_at IS NULL
ORDER BY joined_at, user_id;

-- name: ActiveTeamCount :one
SELECT count(*) FROM team.memberships m JOIN team.teams t ON t.id = m.team_id
WHERE m.user_id = @user_id AND m.left_at IS NULL AND t.status = 'active';

-- name: UserTeamIDs :many
SELECT m.team_id FROM team.memberships m JOIN team.teams t ON t.id = m.team_id
WHERE m.user_id = @user_id AND m.left_at IS NULL AND t.status = 'active'
ORDER BY m.joined_at;

-- name: EndMembership :exec
UPDATE team.memberships SET left_at = @now::timestamptz, leave_reason = @reason::text
WHERE team_id = @team_id AND user_id = @user_id AND left_at IS NULL;

-- name: EndAllMemberships :exec
UPDATE team.memberships SET left_at = @now::timestamptz, leave_reason = 'team_disbanded'
WHERE team_id = @team_id AND left_at IS NULL;

-- name: SetRole :exec
UPDATE team.memberships SET role = @role WHERE team_id = @team_id AND user_id = @user_id AND left_at IS NULL;

-- ---------- Invites ----------

-- name: InsertInvite :exec
INSERT INTO team.invites (id, team_id, invited_user_id, invited_by, created_at, expires_at)
VALUES (@id, @team_id, @invited_user_id, @invited_by, @now, @expires_at);

-- name: GetInviteForUpdate :one
SELECT * FROM team.invites WHERE id = @id FOR UPDATE;

-- name: SetInviteStatus :exec
UPDATE team.invites SET status = @status, responded_at = sqlc.narg(responded_at) WHERE id = @id;

-- Expired pending invites are closed before new ones are counted.
-- name: ExpireInvites :exec
UPDATE team.invites SET status = 'expired' WHERE team_id = @team_id AND status = 'pending' AND expires_at <= @now;

-- name: PendingInvite :one
SELECT EXISTS (SELECT 1 FROM team.invites WHERE team_id = @team_id AND invited_user_id = @user_id AND status = 'pending' AND expires_at > @now);

-- name: CancelInvites :exec
UPDATE team.invites SET status = 'cancelled' WHERE team_id = @team_id AND status = 'pending';

-- name: InvitesOf :many
SELECT i.*, t.name AS team_name, t.tag AS team_tag, t.emblem_id AS team_emblem_id
FROM team.invites i JOIN team.teams t ON t.id = i.team_id
WHERE i.invited_user_id = @user_id AND i.status = 'pending' AND i.expires_at > @now AND t.status = 'active'
ORDER BY i.created_at DESC;

-- ---------- Couples ----------

-- name: CoupleOfPair :one
SELECT * FROM team.couples WHERE user_low_id = @low AND user_high_id = @high FOR UPDATE;

-- name: GetCouple :one
SELECT * FROM team.couples WHERE id = @id;

-- name: GetCoupleForUpdate :one
SELECT * FROM team.couples WHERE id = @id FOR UPDATE;

-- A user's pending or active couple.
-- name: LiveCoupleOf :one
SELECT * FROM team.couples WHERE (user_low_id = @user_id OR user_high_id = @user_id) AND status IN ('pending','active')
LIMIT 1;

-- The last time a user dissolved a couple with someone other than @partner.
-- name: LastDissolvedWithOther :one
SELECT COALESCE(max(last_dissolved_at), 'epoch'::timestamptz)::timestamptz FROM team.couples
WHERE (user_low_id = @user_id OR user_high_id = @user_id) AND NOT (user_low_id = @partner::uuid OR user_high_id = @partner::uuid);

-- name: InsertCouple :exec
INSERT INTO team.couples (id, user_low_id, user_high_id, requested_by, name, created_at, requested_at)
VALUES (@id, @user_low_id, @user_high_id, @requested_by, sqlc.narg(name), @now, @now);

-- name: RequestAgain :exec
UPDATE team.couples SET status = 'pending', requested_by = @requested_by, name = COALESCE(sqlc.narg(name), name),
  requested_at = @now, dissolved_at = NULL
WHERE id = @id;

-- name: SetCoupleStatus :exec
UPDATE team.couples SET status = @status,
  activated_at = CASE WHEN @status = 'active' THEN @now::timestamptz ELSE activated_at END,
  dissolved_at = CASE WHEN @status = 'dissolved' THEN @now::timestamptz ELSE dissolved_at END,
  last_dissolved_at = CASE WHEN @status = 'dissolved' THEN @now::timestamptz ELSE last_dissolved_at END
WHERE id = @id;
