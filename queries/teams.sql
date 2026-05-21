-- name: CreateTeam :one
INSERT INTO teams (id, code, name, flag_url, confederation, tournament_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTeamByID :one
SELECT * FROM teams WHERE id = $1;

-- name: GetTeamByCode :one
SELECT * FROM teams WHERE code = $1 AND tournament_id = $2;

-- name: ListTeamsByTournament :many
SELECT * FROM teams WHERE tournament_id = $1 ORDER BY name;

-- name: CreateExternalID :exec
INSERT INTO team_external_ids (team_id, source, external_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: GetTeamByExternalID :one
SELECT t.*
FROM teams t
JOIN team_external_ids tei ON tei.team_id = t.id
WHERE tei.source = $1 AND tei.external_id = $2;
