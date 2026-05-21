package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
)

// SubmitBracketInput holds the bracket prediction data.
type SubmitBracketInput struct {
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

// SubmitBracketOutput holds the result.
type SubmitBracketOutput struct {
	BracketID string
	UpdatedAt string
}

// SubmitBracket creates or updates a user's bracket prediction.
type SubmitBracket struct {
	brackets    prediction.BracketRepository
	pools       pool.Repository
	tournaments tournament.Repository
}

func NewSubmitBracket(brackets prediction.BracketRepository, pools pool.Repository, tournaments tournament.Repository) *SubmitBracket {
	return &SubmitBracket{brackets: brackets, pools: pools, tournaments: tournaments}
}

func (uc *SubmitBracket) Execute(ctx context.Context, input SubmitBracketInput) (*SubmitBracketOutput, error) {
	userID := shared.UserID(input.UserID)
	poolID := shared.PoolID(input.PoolID)
	tournamentID := shared.TournamentID(input.TournamentID)

	// Verify user is pool member
	members, err := uc.pools.FindMembers(ctx, poolID)
	if err != nil {
		return nil, fmt.Errorf("submit_bracket: pool: %w", err)
	}
	isMember := false
	for _, pm := range members {
		if pm.UserID() == userID {
			isMember = true
			break
		}
	}
	if !isMember {
		return nil, fmt.Errorf("%w: user is not a member of this pool", shared.ErrUnauthorized)
	}

	// Verify tournament exists and bracket is still open
	t, err := uc.tournaments.FindByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("submit_bracket: tournament: %w", err)
	}
	if !t.CanPredictBracket(time.Now()) {
		return nil, fmt.Errorf("%w: bracket predictions are closed (tournament has started)", shared.ErrInvalidInput)
	}

	// Create domain entity (validates coherence)
	bp, err := prediction.NewBracket(
		shared.BracketPredID(uuid.Must(uuid.NewV7()).String()),
		userID, poolID, tournamentID,
		stringSliceToTeamIDs(input.TeamsToRoundOf32),
		stringSliceToTeamIDs(input.TeamsToRoundOf16),
		stringSliceToTeamIDs(input.TeamsToQuarterFinal),
		stringSliceToTeamIDs(input.TeamsToSemiFinal),
		stringSliceToTeamIDs(input.TeamsToFinal),
		shared.TeamID(input.ThirdPlaceWinner),
		shared.TeamID(input.Champion),
	)
	if err != nil {
		return nil, fmt.Errorf("submit_bracket: %w", err)
	}

	if err := uc.brackets.Upsert(ctx, bp); err != nil {
		return nil, fmt.Errorf("submit_bracket: upsert: %w", err)
	}

	return &SubmitBracketOutput{
		BracketID: string(bp.ID()),
	}, nil
}

func stringSliceToTeamIDs(ss []string) []shared.TeamID {
	res := make([]shared.TeamID, len(ss))
	for i, s := range ss {
		res[i] = shared.TeamID(s)
	}
	return res
}
