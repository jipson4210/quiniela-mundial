package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/josemontalban/quiniela-mundial/internal/domain/scoring"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/persistence/postgres/sqlc"
)

type ScoreRepo struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

func NewScoreRepo(db *pgxpool.Pool) *ScoreRepo {
	return &ScoreRepo{q: sqlc.New(db), db: db}
}

func (r *ScoreRepo) Upsert(ctx context.Context, se *scoring.ScoreEntry) error {
	_, err := r.q.UpsertScore(ctx, sqlc.UpsertScoreParams{
		ID:         string(se.ID()),
		UserID:     string(se.UserID()),
		PoolID:     string(se.PoolID()),
		SourceType: string(se.SourceType()),
		SourceRef:  se.SourceRef(),
		Points:     se.Points(),
	})
	return err
}

func (r *ScoreRepo) FindByUserAndPool(ctx context.Context, userID shared.UserID, poolID shared.PoolID) ([]*scoring.ScoreEntry, error) {
	rows, err := r.q.GetScoresByUserAndPool(ctx, string(userID), string(poolID))
	if err != nil {
		return nil, err
	}
	entries := make([]*scoring.ScoreEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, toScoreEntryDomain(row))
	}
	return entries, nil
}

func (r *ScoreRepo) FindByPool(ctx context.Context, poolID shared.PoolID) ([]*scoring.ScoreEntry, error) {
	rows, err := r.q.GetScoresByPool(ctx, string(poolID))
	if err != nil {
		return nil, err
	}
	entries := make([]*scoring.ScoreEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, toScoreEntryDomain(row))
	}
	return entries, nil
}

func (r *ScoreRepo) SumByUserAndPool(ctx context.Context, userID shared.UserID, poolID shared.PoolID) (int, error) {
	sum, err := r.q.SumScoresByUserAndPool(ctx, string(userID), string(poolID))
	return int(sum), err
}

func (r *ScoreRepo) DeleteBySourceRef(ctx context.Context, sourceType scoring.SourceType, sourceRef string) error {
	return r.q.DeleteScoresBySourceRef(ctx, string(sourceType), sourceRef)
}

func toScoreEntryDomain(row sqlc.ScoreEntry) *scoring.ScoreEntry {
	return scoring.ReconstructScoreEntry(
		shared.ScoreEntryID(row.ID),
		shared.UserID(row.UserID),
		shared.PoolID(row.PoolID),
		scoring.SourceType(row.SourceType),
		row.SourceRef,
		row.Points,
		row.ComputedAt,
		row.Version,
	)
}
