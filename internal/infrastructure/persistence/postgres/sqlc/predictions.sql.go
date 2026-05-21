package sqlc

import (
	"context"
	"time"
)

// MatchPrediction represents a row from match_predictions.
type MatchPrediction struct {
	ID          string
	UserID      string
	PoolID      string
	MatchID     string
	HomeGoals   int
	AwayGoals   int
	SubmittedAt time.Time
	UpdatedAt   time.Time
}

type UpsertPredictionParams struct {
	ID        string
	UserID    string
	PoolID    string
	MatchID   string
	HomeGoals int
	AwayGoals int
}

func (q *Queries) UpsertPrediction(ctx context.Context, arg UpsertPredictionParams) (MatchPrediction, error) {
	const sql = `INSERT INTO match_predictions (id, user_id, pool_id, match_id, home_goals, away_goals)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, pool_id, match_id)
		DO UPDATE SET home_goals = $5, away_goals = $6, updated_at = now()
		RETURNING id, user_id, pool_id, match_id, home_goals, away_goals, submitted_at, updated_at`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.UserID, arg.PoolID, arg.MatchID, arg.HomeGoals, arg.AwayGoals)
	var p MatchPrediction
	err := row.Scan(&p.ID, &p.UserID, &p.PoolID, &p.MatchID, &p.HomeGoals, &p.AwayGoals, &p.SubmittedAt, &p.UpdatedAt)
	return p, err
}

func (q *Queries) GetPredictionByUserAndMatch(ctx context.Context, userID, poolID, matchID string) (MatchPrediction, error) {
	const sql = `SELECT id, user_id, pool_id, match_id, home_goals, away_goals, submitted_at, updated_at
		FROM match_predictions WHERE user_id = $1 AND pool_id = $2 AND match_id = $3`
	row := q.db.QueryRow(ctx, sql, userID, poolID, matchID)
	var p MatchPrediction
	err := row.Scan(&p.ID, &p.UserID, &p.PoolID, &p.MatchID, &p.HomeGoals, &p.AwayGoals, &p.SubmittedAt, &p.UpdatedAt)
	return p, err
}

func (q *Queries) GetPredictionsByPoolAndMatch(ctx context.Context, poolID, matchID string) ([]MatchPrediction, error) {
	const sql = `SELECT id, user_id, pool_id, match_id, home_goals, away_goals, submitted_at, updated_at
		FROM match_predictions WHERE pool_id = $1 AND match_id = $2 ORDER BY updated_at DESC`
	rows, err := q.db.Query(ctx, sql, poolID, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var preds []MatchPrediction
	for rows.Next() {
		var p MatchPrediction
		if err := rows.Scan(&p.ID, &p.UserID, &p.PoolID, &p.MatchID, &p.HomeGoals, &p.AwayGoals, &p.SubmittedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

func (q *Queries) GetPredictionsByUserAndPool(ctx context.Context, userID, poolID string) ([]MatchPrediction, error) {
	const sql = `SELECT id, user_id, pool_id, match_id, home_goals, away_goals, submitted_at, updated_at
		FROM match_predictions WHERE user_id = $1 AND pool_id = $2 ORDER BY updated_at DESC`
	rows, err := q.db.Query(ctx, sql, userID, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var preds []MatchPrediction
	for rows.Next() {
		var p MatchPrediction
		if err := rows.Scan(&p.ID, &p.UserID, &p.PoolID, &p.MatchID, &p.HomeGoals, &p.AwayGoals, &p.SubmittedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	return preds, rows.Err()
}
