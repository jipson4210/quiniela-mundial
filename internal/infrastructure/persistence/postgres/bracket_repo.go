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

type BracketRepo struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

func NewBracketRepo(db *pgxpool.Pool) *BracketRepo {
	return &BracketRepo{q: sqlc.New(db), db: db}
}

func (r *BracketRepo) Upsert(ctx context.Context, bp *prediction.BracketPrediction) error {
	_, err := r.q.UpsertBracket(ctx, sqlc.UpsertBracketParams{
		ID:                  string(bp.ID()),
		UserID:              string(bp.UserID()),
		PoolID:              string(bp.PoolID()),
		TournamentID:        string(bp.TournamentID()),
		TeamsToRoundOf32:    teamIDsToStrings(bp.TeamsToRoundOf32()),
		TeamsToRoundOf16:    teamIDsToStrings(bp.TeamsToRoundOf16()),
		TeamsToQuarterFinal: teamIDsToStrings(bp.TeamsToQuarterFinal()),
		TeamsToSemiFinal:    teamIDsToStrings(bp.TeamsToSemiFinal()),
		TeamsToFinal:        teamIDsToStrings(bp.TeamsToFinal()),
		ThirdPlaceWinner:    string(bp.ThirdPlaceWinner()),
		Champion:            string(bp.Champion()),
	})
	return err
}

func (r *BracketRepo) FindByUserAndPool(ctx context.Context, userID shared.UserID, poolID shared.PoolID) (*prediction.BracketPrediction, error) {
	row, err := r.q.GetBracketByUserAndPool(ctx, string(userID), string(poolID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toBracketDomain(row), nil
}

func teamIDsToStrings(ids []shared.TeamID) []string {
	res := make([]string, len(ids))
	for i, id := range ids {
		res[i] = string(id)
	}
	return res
}

func toBracketDomain(row sqlc.BracketPrediction) *prediction.BracketPrediction {
	return prediction.ReconstructBracket(
		shared.BracketPredID(row.ID),
		shared.UserID(row.UserID),
		shared.PoolID(row.PoolID),
		shared.TournamentID(row.TournamentID),
		stringSliceToTeamIDs(row.TeamsToRoundOf32),
		stringSliceToTeamIDs(row.TeamsToRoundOf16),
		stringSliceToTeamIDs(row.TeamsToQuarterFinal),
		stringSliceToTeamIDs(row.TeamsToSemiFinal),
		stringSliceToTeamIDs(row.TeamsToFinal),
		shared.TeamID(row.ThirdPlaceWinner),
		shared.TeamID(row.Champion),
		row.SubmittedAt,
		row.UpdatedAt,
	)
}

func stringSliceToTeamIDs(ss []string) []shared.TeamID {
	res := make([]shared.TeamID, len(ss))
	for i, s := range ss {
		res[i] = shared.TeamID(s)
	}
	return res
}
