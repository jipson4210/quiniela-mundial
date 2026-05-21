package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/team"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/persistence/postgres/sqlc"
)

type TeamRepo struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

func NewTeamRepo(db *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{q: sqlc.New(db), db: db}
}

func (r *TeamRepo) Save(ctx context.Context, t *team.Team) error {
	_, err := r.q.CreateTeam(ctx, sqlc.CreateTeamParams{
		ID:            string(t.ID()),
		Code:          t.Code(),
		Name:          t.Name(),
		FlagUrl:       t.FlagURL(),
		Confederation: t.Confederation(),
		TournamentID:  string(t.TournamentID()),
	})
	return err
}

func (r *TeamRepo) SaveBatch(ctx context.Context, teams []*team.Team) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)
	for _, t := range teams {
		_, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{
			ID:            string(t.ID()),
			Code:          t.Code(),
			Name:          t.Name(),
			FlagUrl:       t.FlagURL(),
			Confederation: t.Confederation(),
			TournamentID:  string(t.TournamentID()),
		})
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *TeamRepo) FindByID(ctx context.Context, id shared.TeamID) (*team.Team, error) {
	row, err := r.q.GetTeamByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toTeamDomain(row), nil
}

func (r *TeamRepo) FindByCode(ctx context.Context, code string, tournamentID shared.TournamentID) (*team.Team, error) {
	row, err := r.q.GetTeamByCode(ctx, sqlc.GetTeamByCodeParams{
		Code:         code,
		TournamentID: string(tournamentID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toTeamDomain(row), nil
}

func (r *TeamRepo) FindByTournament(ctx context.Context, tournamentID shared.TournamentID) ([]*team.Team, error) {
	rows, err := r.q.ListTeamsByTournament(ctx, string(tournamentID))
	if err != nil {
		return nil, err
	}
	teams := make([]*team.Team, 0, len(rows))
	for _, row := range rows {
		teams = append(teams, toTeamDomain(row))
	}
	return teams, nil
}

func (r *TeamRepo) SaveExternalID(ctx context.Context, e team.ExternalID) error {
	return r.q.CreateExternalID(ctx, sqlc.CreateExternalIDParams{
		TeamID:     string(e.TeamID()),
		Source:     e.Source(),
		ExternalID: e.ExternalID(),
	})
}

func (r *TeamRepo) FindByExternalID(ctx context.Context, source, externalID string) (*team.Team, error) {
	row, err := r.q.GetTeamByExternalID(ctx, sqlc.GetTeamByExternalIDParams{
		Source:     source,
		ExternalID: externalID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toTeamDomain(row), nil
}

func toTeamDomain(row sqlc.Team) *team.Team {
	return team.Reconstruct(
		shared.TeamID(row.ID),
		row.Code,
		row.Name,
		stringOrEmpty(row.FlagUrl),
		stringOrEmpty(row.Confederation),
		shared.TournamentID(row.TournamentID),
	)
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
