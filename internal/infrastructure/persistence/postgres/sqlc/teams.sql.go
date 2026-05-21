package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Team struct {
	ID            string
	Code          string
	Name          string
	FlagUrl       *string
	Confederation *string
	TournamentID  string
}

type CreateTeamParams struct {
	ID            string
	Code          string
	Name          string
	FlagUrl       string
	Confederation string
	TournamentID  string
}

func (q *Queries) CreateTeam(ctx context.Context, arg CreateTeamParams) (Team, error) {
	const sql = `INSERT INTO teams (id, code, name, flag_url, confederation, tournament_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, code, name, flag_url, confederation, tournament_id`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.Code, arg.Name, strOrNil(arg.FlagUrl), strOrNil(arg.Confederation), arg.TournamentID)
	var t Team
	err := row.Scan(&t.ID, &t.Code, &t.Name, &t.FlagUrl, &t.Confederation, &t.TournamentID)
	return t, err
}

func (q *Queries) GetTeamByID(ctx context.Context, id string) (Team, error) {
	const sql = `SELECT id, code, name, flag_url, confederation, tournament_id FROM teams WHERE id = $1`
	row := q.db.QueryRow(ctx, sql, id)
	var t Team
	err := row.Scan(&t.ID, &t.Code, &t.Name, &t.FlagUrl, &t.Confederation, &t.TournamentID)
	if err == pgx.ErrNoRows {
		return Team{}, err
	}
	return t, err
}

type GetTeamByCodeParams struct {
	Code         string
	TournamentID string
}

func (q *Queries) GetTeamByCode(ctx context.Context, arg GetTeamByCodeParams) (Team, error) {
	const sql = `SELECT id, code, name, flag_url, confederation, tournament_id FROM teams WHERE code = $1 AND tournament_id = $2`
	row := q.db.QueryRow(ctx, sql, arg.Code, arg.TournamentID)
	var t Team
	err := row.Scan(&t.ID, &t.Code, &t.Name, &t.FlagUrl, &t.Confederation, &t.TournamentID)
	if err == pgx.ErrNoRows {
		return Team{}, err
	}
	return t, err
}

func (q *Queries) ListTeamsByTournament(ctx context.Context, tournamentID string) ([]Team, error) {
	const sql = `SELECT id, code, name, flag_url, confederation, tournament_id FROM teams WHERE tournament_id = $1 ORDER BY name`
	rows, err := q.db.Query(ctx, sql, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.FlagUrl, &t.Confederation, &t.TournamentID); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

type CreateExternalIDParams struct {
	TeamID     string
	Source     string
	ExternalID string
}

func (q *Queries) CreateExternalID(ctx context.Context, arg CreateExternalIDParams) error {
	const sql = `INSERT INTO team_external_ids (team_id, source, external_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	_, err := q.db.Exec(ctx, sql, arg.TeamID, arg.Source, arg.ExternalID)
	return err
}

type GetTeamByExternalIDParams struct {
	Source     string
	ExternalID string
}

func (q *Queries) GetTeamByExternalID(ctx context.Context, arg GetTeamByExternalIDParams) (Team, error) {
	const sql = `SELECT t.id, t.code, t.name, t.flag_url, t.confederation, t.tournament_id
		FROM teams t JOIN team_external_ids tei ON tei.team_id = t.id
		WHERE tei.source = $1 AND tei.external_id = $2`
	row := q.db.QueryRow(ctx, sql, arg.Source, arg.ExternalID)
	var t Team
	err := row.Scan(&t.ID, &t.Code, &t.Name, &t.FlagUrl, &t.Confederation, &t.TournamentID)
	return t, err
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
