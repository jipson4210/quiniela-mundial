package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Match struct {
	ID            string
	TournamentID  string
	Stage         string
	GroupID       *string
	HomeTeamID    string
	AwayTeamID    string
	KickoffAt     time.Time
	Venue         string
	Status        string
	HomeGoals     *int
	AwayGoals     *int
	HomeGoalsEt   *int
	AwayGoalsEt   *int
	HomeGoalsPen  *int
	AwayGoalsPen  *int
	ResultSource  *string
	FinalizedAt   *time.Time
	CreatedAt     time.Time
}

type CreateMatchParams struct {
	ID           string
	TournamentID string
	Stage        string
	GroupID      *string
	HomeTeamID   string
	AwayTeamID   string
	KickoffAt    time.Time
	Venue        string
	Status       string
}

func (q *Queries) CreateMatch(ctx context.Context, arg CreateMatchParams) (Match, error) {
	const sql = `INSERT INTO matches (id, tournament_id, stage, group_id, home_team_id, away_team_id, kickoff_at, venue, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tournament_id, stage, group_id, home_team_id, away_team_id, kickoff_at, venue, status,
			home_goals, away_goals, home_goals_et, away_goals_et, home_goals_pen, away_goals_pen, result_source, finalized_at, created_at`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.TournamentID, arg.Stage, arg.GroupID, arg.HomeTeamID, arg.AwayTeamID, arg.KickoffAt, arg.Venue, arg.Status)
	var m Match
	err := row.Scan(&m.ID, &m.TournamentID, &m.Stage, &m.GroupID, &m.HomeTeamID, &m.AwayTeamID,
		&m.KickoffAt, &m.Venue, &m.Status,
		&m.HomeGoals, &m.AwayGoals, &m.HomeGoalsEt, &m.AwayGoalsEt, &m.HomeGoalsPen, &m.AwayGoalsPen,
		&m.ResultSource, &m.FinalizedAt, &m.CreatedAt)
	return m, err
}

func (q *Queries) GetMatchByID(ctx context.Context, id string) (Match, error) {
	const sql = `SELECT id, tournament_id, stage, group_id, home_team_id, away_team_id, kickoff_at, venue, status,
		home_goals, away_goals, home_goals_et, away_goals_et, home_goals_pen, away_goals_pen, result_source, finalized_at, created_at
		FROM matches WHERE id = $1`
	row := q.db.QueryRow(ctx, sql, id)
	var m Match
	err := row.Scan(&m.ID, &m.TournamentID, &m.Stage, &m.GroupID, &m.HomeTeamID, &m.AwayTeamID,
		&m.KickoffAt, &m.Venue, &m.Status,
		&m.HomeGoals, &m.AwayGoals, &m.HomeGoalsEt, &m.AwayGoalsEt, &m.HomeGoalsPen, &m.AwayGoalsPen,
		&m.ResultSource, &m.FinalizedAt, &m.CreatedAt)
	if err == pgx.ErrNoRows {
		return Match{}, err
	}
	return m, err
}

func (q *Queries) ListMatchesByTournament(ctx context.Context, tournamentID string) ([]Match, error) {
	const sql = `SELECT id, tournament_id, stage, group_id, home_team_id, away_team_id, kickoff_at, venue, status,
		home_goals, away_goals, home_goals_et, away_goals_et, home_goals_pen, away_goals_pen, result_source, finalized_at, created_at
		FROM matches WHERE tournament_id = $1 ORDER BY kickoff_at`
	rows, err := q.db.Query(ctx, sql, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatches(rows)
}

type ListMatchesByStageParams struct {
	TournamentID string
	Stage        string
}

func (q *Queries) ListMatchesByStage(ctx context.Context, arg ListMatchesByStageParams) ([]Match, error) {
	const sql = `SELECT id, tournament_id, stage, group_id, home_team_id, away_team_id, kickoff_at, venue, status,
		home_goals, away_goals, home_goals_et, away_goals_et, home_goals_pen, away_goals_pen, result_source, finalized_at, created_at
		FROM matches WHERE tournament_id = $1 AND stage = $2 ORDER BY kickoff_at`
	rows, err := q.db.Query(ctx, sql, arg.TournamentID, arg.Stage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatches(rows)
}

func (q *Queries) GetMatchByTeamsAndKickoff(ctx context.Context, homeTeamID, awayTeamID string, kickoffAt time.Time) (Match, error) {
	const sql = `SELECT id, tournament_id, stage, group_id, home_team_id, away_team_id, kickoff_at, venue, status,
		home_goals, away_goals, home_goals_et, away_goals_et, home_goals_pen, away_goals_pen, result_source, finalized_at, created_at
		FROM matches WHERE home_team_id = $1 AND away_team_id = $2 AND kickoff_at = $3`
	row := q.db.QueryRow(ctx, sql, homeTeamID, awayTeamID, kickoffAt)
	var m Match
	err := row.Scan(&m.ID, &m.TournamentID, &m.Stage, &m.GroupID, &m.HomeTeamID, &m.AwayTeamID,
		&m.KickoffAt, &m.Venue, &m.Status,
		&m.HomeGoals, &m.AwayGoals, &m.HomeGoalsEt, &m.AwayGoalsEt, &m.HomeGoalsPen, &m.AwayGoalsPen,
		&m.ResultSource, &m.FinalizedAt, &m.CreatedAt)
	return m, err
}

type UpdateMatchResultParams struct {
	ID           string
	HomeGoals    int
	AwayGoals    int
	HomeGoalsEt  *int
	AwayGoalsEt  *int
	HomeGoalsPen *int
	AwayGoalsPen *int
	ResultSource string
	FinalizedAt  time.Time
}

func (q *Queries) UpdateMatchResult(ctx context.Context, arg UpdateMatchResultParams) error {
	const sql = `UPDATE matches SET status = 'finished', home_goals = $2, away_goals = $3,
		home_goals_et = $4, away_goals_et = $5, home_goals_pen = $6, away_goals_pen = $7,
		result_source = $8, finalized_at = $9 WHERE id = $1`
	_, err := q.db.Exec(ctx, sql, arg.ID, arg.HomeGoals, arg.AwayGoals,
		arg.HomeGoalsEt, arg.AwayGoalsEt, arg.HomeGoalsPen, arg.AwayGoalsPen,
		arg.ResultSource, arg.FinalizedAt)
	return err
}

type CreateStageGroupParams struct {
	ID           string
	TournamentID string
	Name         string
}

func (q *Queries) CreateStageGroup(ctx context.Context, arg CreateStageGroupParams) (interface{}, error) {
	const sql = `INSERT INTO stage_groups (id, tournament_id, name) VALUES ($1, $2, $3) RETURNING id, tournament_id, name`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.TournamentID, arg.Name)
	var id, tid, name string
	err := row.Scan(&id, &tid, &name)
	return map[string]string{"id": id, "tournament_id": tid, "name": name}, err
}

type AddTeamToGroupParams struct {
	GroupID string
	TeamID  string
}

func (q *Queries) AddTeamToGroup(ctx context.Context, arg AddTeamToGroupParams) error {
	const sql = `INSERT INTO stage_group_teams (group_id, team_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := q.db.Exec(ctx, sql, arg.GroupID, arg.TeamID)
	return err
}

type CreateStageParams struct {
	ID                   string
	TournamentID         string
	Stage                string
	PointsPerCorrectTeam int
	Description          *string
}

func (q *Queries) CreateStage(ctx context.Context, arg CreateStageParams) (interface{}, error) {
	const sql = `INSERT INTO stages (id, tournament_id, stage, points_per_correct_team, description)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, tournament_id, stage, points_per_correct_team, description`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.TournamentID, arg.Stage, arg.PointsPerCorrectTeam, arg.Description)
	var id, tid, stage, desc string
	var pts int
	err := row.Scan(&id, &tid, &stage, &pts, &desc)
	return map[string]interface{}{"id": id}, err
}

func (q *Queries) CountMatchesByTournament(ctx context.Context, tournamentID string) (int64, error) {
	const sql = `SELECT COUNT(*) FROM matches WHERE tournament_id = $1`
	row := q.db.QueryRow(ctx, sql, tournamentID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) CountTeamsByTournament(ctx context.Context, tournamentID string) (int64, error) {
	const sql = `SELECT COUNT(*) FROM teams WHERE tournament_id = $1`
	row := q.db.QueryRow(ctx, sql, tournamentID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

func scanMatches(rows pgx.Rows) ([]Match, error) {
	var matches []Match
	for rows.Next() {
		var m Match
		if err := rows.Scan(&m.ID, &m.TournamentID, &m.Stage, &m.GroupID, &m.HomeTeamID, &m.AwayTeamID,
			&m.KickoffAt, &m.Venue, &m.Status,
			&m.HomeGoals, &m.AwayGoals, &m.HomeGoalsEt, &m.AwayGoalsEt, &m.HomeGoalsPen, &m.AwayGoalsPen,
			&m.ResultSource, &m.FinalizedAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}
