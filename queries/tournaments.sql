-- name: CreateTournament :one
INSERT INTO tournaments (id, name, starts_at, ends_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTournamentByID :one
SELECT * FROM tournaments WHERE id = $1;

-- name: GetCurrentTournament :one
SELECT * FROM tournaments ORDER BY starts_at DESC LIMIT 1;

-- name: GetTournaments :many
SELECT * FROM tournaments ORDER BY starts_at DESC;
