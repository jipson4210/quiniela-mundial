package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type ScoreEntry struct {
	ID          string
	UserID      string
	PoolID      string
	SourceType  string
	SourceRef   string
	Points      int
	ComputedAt  time.Time
	Version     int
}

type UpsertScoreParams struct {
	ID         string
	UserID     string
	PoolID     string
	SourceType string
	SourceRef  string
	Points     int
}

func (q *Queries) UpsertScore(ctx context.Context, arg UpsertScoreParams) (ScoreEntry, error) {
	const sql = `INSERT INTO score_entries (id, user_id, pool_id, source_type, source_ref, points)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, pool_id, source_type, source_ref)
		DO UPDATE SET points = $6, version = score_entries.version + 1, computed_at = now()
		RETURNING id, user_id, pool_id, source_type, source_ref, points, computed_at, version`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.UserID, arg.PoolID, arg.SourceType, arg.SourceRef, arg.Points)
	var se ScoreEntry
	err := row.Scan(&se.ID, &se.UserID, &se.PoolID, &se.SourceType, &se.SourceRef, &se.Points, &se.ComputedAt, &se.Version)
	return se, err
}

func (q *Queries) GetScoresByUserAndPool(ctx context.Context, userID, poolID string) ([]ScoreEntry, error) {
	const sql = `SELECT id, user_id, pool_id, source_type, source_ref, points, computed_at, version
		FROM score_entries WHERE user_id = $1 AND pool_id = $2 ORDER BY computed_at DESC`
	rows, err := q.db.Query(ctx, sql, userID, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScores(rows)
}

func (q *Queries) GetScoresByPool(ctx context.Context, poolID string) ([]ScoreEntry, error) {
	const sql = `SELECT id, user_id, pool_id, source_type, source_ref, points, computed_at, version
		FROM score_entries WHERE pool_id = $1 ORDER BY points DESC`
	rows, err := q.db.Query(ctx, sql, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScores(rows)
}

func (q *Queries) SumScoresByUserAndPool(ctx context.Context, userID, poolID string) (int64, error) {
	const sql = `SELECT COALESCE(SUM(points), 0) FROM score_entries WHERE user_id = $1 AND pool_id = $2`
	row := q.db.QueryRow(ctx, sql, userID, poolID)
	var sum int64
	err := row.Scan(&sum)
	return sum, err
}

func (q *Queries) DeleteScoresBySourceRef(ctx context.Context, sourceType, sourceRef string) error {
	const sql = `DELETE FROM score_entries WHERE source_type = $1 AND source_ref = $2`
	_, err := q.db.Exec(ctx, sql, sourceType, sourceRef)
	return err
}

func scanScores(rows pgx.Rows) ([]ScoreEntry, error) {
	var entries []ScoreEntry
	for rows.Next() {
		var se ScoreEntry
		if err := rows.Scan(&se.ID, &se.UserID, &se.PoolID, &se.SourceType, &se.SourceRef, &se.Points, &se.ComputedAt, &se.Version); err != nil {
			return nil, err
		}
		entries = append(entries, se)
	}
	return entries, rows.Err()
}
