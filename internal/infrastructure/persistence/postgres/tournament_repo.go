package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/persistence/postgres/sqlc"
)

type TournamentRepo struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

func NewTournamentRepo(db *pgxpool.Pool) *TournamentRepo {
	return &TournamentRepo{q: sqlc.New(db), db: db}
}

func (r *TournamentRepo) Save(ctx context.Context, t *tournament.Tournament) error {
	row, err := r.q.CreateTournament(ctx, sqlc.CreateTournamentParams{
		ID:       string(t.ID()),
		Name:     t.Name(),
		StartsAt: t.StartsAt(),
		EndsAt:   t.EndsAt(),
	})
	if err != nil {
		return err
	}
	_ = row
	return nil
}

func (r *TournamentRepo) FindByID(ctx context.Context, id shared.TournamentID) (*tournament.Tournament, error) {
	row, err := r.q.GetTournamentByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toTournamentDomain(row), nil
}

func (r *TournamentRepo) FindCurrent(ctx context.Context) (*tournament.Tournament, error) {
	row, err := r.q.GetCurrentTournament(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toTournamentDomain(row), nil
}

// toTournamentDomain maps sqlc row to domain Tournament (lightweight — groups/stages loaded separately)
func toTournamentDomain(row sqlc.Tournament) *tournament.Tournament {
	return tournament.Reconstruct(
		shared.TournamentID(row.ID),
		row.Name,
		row.StartsAt,
		row.EndsAt,
		row.CreatedAt,
		nil, // groups loaded on demand
		nil, // stages loaded on demand
		nil, // teams loaded on demand
	)
}
