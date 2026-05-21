package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type BracketPrediction struct {
	ID                  string
	UserID              string
	PoolID              string
	TournamentID        string
	TeamsToRoundOf32    []string
	TeamsToRoundOf16    []string
	TeamsToQuarterFinal []string
	TeamsToSemiFinal    []string
	TeamsToFinal        []string
	ThirdPlaceWinner    string
	Champion            string
	SubmittedAt         time.Time
	UpdatedAt           time.Time
}

type UpsertBracketParams struct {
	ID                  string
	UserID              string
	PoolID              string
	TournamentID        string
	TeamsToRoundOf32    []string
	TeamsToRoundOf16    []string
	TeamsToQuarterFinal []string
	TeamsToSemiFinal    []string
	TeamsToFinal        []string
	ThirdPlaceWinner    string
	Champion            string
}

func (q *Queries) UpsertBracket(ctx context.Context, arg UpsertBracketParams) (BracketPrediction, error) {
	const sql = `INSERT INTO bracket_predictions (
			id, user_id, pool_id, tournament_id,
			teams_to_round_of_32, teams_to_round_of_16,
			teams_to_quarter_final, teams_to_semi_final,
			teams_to_final, third_place_winner, champion
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, pool_id)
		DO UPDATE SET
			tournament_id = $4,
			teams_to_round_of_32 = $5,
			teams_to_round_of_16 = $6,
			teams_to_quarter_final = $7,
			teams_to_semi_final = $8,
			teams_to_final = $9,
			third_place_winner = $10,
			champion = $11,
			updated_at = now()
		RETURNING id, user_id, pool_id, tournament_id,
			teams_to_round_of_32, teams_to_round_of_16,
			teams_to_quarter_final, teams_to_semi_final,
			teams_to_final, third_place_winner, champion,
			submitted_at, updated_at`
	row := q.db.QueryRow(ctx, sql,
		arg.ID, arg.UserID, arg.PoolID, arg.TournamentID,
		arg.TeamsToRoundOf32, arg.TeamsToRoundOf16,
		arg.TeamsToQuarterFinal, arg.TeamsToSemiFinal,
		arg.TeamsToFinal, arg.ThirdPlaceWinner, arg.Champion,
	)
	var bp BracketPrediction
	err := row.Scan(&bp.ID, &bp.UserID, &bp.PoolID, &bp.TournamentID,
		&bp.TeamsToRoundOf32, &bp.TeamsToRoundOf16,
		&bp.TeamsToQuarterFinal, &bp.TeamsToSemiFinal,
		&bp.TeamsToFinal, &bp.ThirdPlaceWinner, &bp.Champion,
		&bp.SubmittedAt, &bp.UpdatedAt)
	return bp, err
}

func (q *Queries) GetBracketByUserAndPool(ctx context.Context, userID, poolID string) (BracketPrediction, error) {
	const sql = `SELECT id, user_id, pool_id, tournament_id,
		teams_to_round_of_32, teams_to_round_of_16,
		teams_to_quarter_final, teams_to_semi_final,
		teams_to_final, third_place_winner, champion,
		submitted_at, updated_at
		FROM bracket_predictions WHERE user_id = $1 AND pool_id = $2`
	row := q.db.QueryRow(ctx, sql, userID, poolID)
	var bp BracketPrediction
	err := row.Scan(&bp.ID, &bp.UserID, &bp.PoolID, &bp.TournamentID,
		&bp.TeamsToRoundOf32, &bp.TeamsToRoundOf16,
		&bp.TeamsToQuarterFinal, &bp.TeamsToSemiFinal,
		&bp.TeamsToFinal, &bp.ThirdPlaceWinner, &bp.Champion,
		&bp.SubmittedAt, &bp.UpdatedAt)
	if err == pgx.ErrNoRows {
		return BracketPrediction{}, err
	}
	return bp, err
}
