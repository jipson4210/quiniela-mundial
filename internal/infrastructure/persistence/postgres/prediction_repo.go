package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/persistence/postgres/sqlc"
)

type PredictionRepo struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

func NewPredictionRepo(db *pgxpool.Pool) *PredictionRepo {
	return &PredictionRepo{q: sqlc.New(db), db: db}
}

func (r *PredictionRepo) Upsert(ctx context.Context, p *prediction.MatchPrediction) error {
	_, err := r.q.UpsertPrediction(ctx, sqlc.UpsertPredictionParams{
		ID:        string(p.ID()),
		UserID:    string(p.UserID()),
		PoolID:    string(p.PoolID()),
		MatchID:   string(p.MatchID()),
		HomeGoals: p.HomeGoals(),
		AwayGoals: p.AwayGoals(),
	})
	return err
}

func (r *PredictionRepo) FindByUserAndMatch(ctx context.Context, userID shared.UserID, poolID shared.PoolID, matchID shared.MatchID) (*prediction.MatchPrediction, error) {
	row, err := r.q.GetPredictionByUserAndMatch(ctx, string(userID), string(poolID), string(matchID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toPredictionDomain(row), nil
}

func (r *PredictionRepo) FindByPoolAndMatch(ctx context.Context, poolID shared.PoolID, matchID shared.MatchID) ([]*prediction.MatchPrediction, error) {
	rows, err := r.q.GetPredictionsByPoolAndMatch(ctx, string(poolID), string(matchID))
	if err != nil {
		return nil, err
	}
	preds := make([]*prediction.MatchPrediction, 0, len(rows))
	for _, row := range rows {
		preds = append(preds, toPredictionDomain(row))
	}
	return preds, nil
}

func (r *PredictionRepo) FindByUserAndPool(ctx context.Context, userID shared.UserID, poolID shared.PoolID) ([]*prediction.MatchPrediction, error) {
	rows, err := r.q.GetPredictionsByUserAndPool(ctx, string(userID), string(poolID))
	if err != nil {
		return nil, err
	}
	preds := make([]*prediction.MatchPrediction, 0, len(rows))
	for _, row := range rows {
		preds = append(preds, toPredictionDomain(row))
	}
	return preds, nil
}

func (r *PredictionRepo) FindDistinctPoolsByMatch(ctx context.Context, matchID shared.MatchID) ([]shared.PoolID, error) {
	ids, err := r.q.GetDistinctPoolsByMatch(ctx, string(matchID))
	if err != nil {
		return nil, err
	}
	poolIDs := make([]shared.PoolID, len(ids))
	for i, id := range ids {
		poolIDs[i] = shared.PoolID(id)
	}
	return poolIDs, nil
}

func toPredictionDomain(row sqlc.MatchPrediction) *prediction.MatchPrediction {
	return prediction.Reconstruct(
		shared.PredictionID(row.ID),
		shared.UserID(row.UserID),
		shared.PoolID(row.PoolID),
		shared.MatchID(row.MatchID),
		row.HomeGoals,
		row.AwayGoals,
		row.SubmittedAt,
		row.UpdatedAt,
	)
}
