-- name: CreateMatch :one
INSERT INTO matches (
    id, tournament_id, stage, group_id,
    home_team_id, away_team_id, kickoff_at, venue, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetMatchByID :one
SELECT * FROM matches WHERE id = $1;

-- name: ListMatchesByTournament :many
SELECT * FROM matches WHERE tournament_id = $1 ORDER BY kickoff_at;

-- name: ListMatchesByStage :many
SELECT * FROM matches
WHERE tournament_id = $1 AND stage = $2
ORDER BY kickoff_at;

-- name: GetMatchByTeamsAndKickoff :one
SELECT * FROM matches
WHERE home_team_id = $1 AND away_team_id = $2 AND kickoff_at = $3;

-- name: UpdateMatchResult :exec
UPDATE matches SET
    status = 'finished',
    home_goals = $2,
    away_goals = $3,
    home_goals_et = $4,
    away_goals_et = $5,
    home_goals_pen = $6,
    away_goals_pen = $7,
    result_source = $8,
    finalized_at = $9
WHERE id = $1;

-- name: CreateStageGroup :one
INSERT INTO stage_groups (id, tournament_id, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: AddTeamToGroup :exec
INSERT INTO stage_group_teams (group_id, team_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: CreateStage :one
INSERT INTO stages (id, tournament_id, stage, points_per_correct_team, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CountMatchesByTournament :one
SELECT COUNT(*) FROM matches WHERE tournament_id = $1;

-- name: CountTeamsByTournament :one
SELECT COUNT(*) FROM teams WHERE tournament_id = $1;
