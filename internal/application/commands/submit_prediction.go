package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// SubmitPredictionInput holds the data to submit or update a match prediction.
type SubmitPredictionInput struct {
	UserID    string
	PoolID    string
	MatchID   string
	HomeGoals int
	AwayGoals int
}

// SubmitPredictionOutput holds the result.
type SubmitPredictionOutput struct {
	PredictionID string
	HomeGoals    int
	AwayGoals    int
	UpdatedAt    string
}

// SubmitPrediction creates or updates a match prediction, validating the cutoff.
type SubmitPrediction struct {
	predictions prediction.Repository
	matches     match.Repository
	pools       pool.Repository
}

func NewSubmitPrediction(predictions prediction.Repository, matches match.Repository, pools pool.Repository) *SubmitPrediction {
	return &SubmitPrediction{predictions: predictions, matches: matches, pools: pools}
}

func (uc *SubmitPrediction) Execute(ctx context.Context, input SubmitPredictionInput) (*SubmitPredictionOutput, error) {
	// Verify match exists and is still open for predictions
	m, err := uc.matches.FindByID(ctx, shared.MatchID(input.MatchID))
	if err != nil {
		return nil, fmt.Errorf("submit_prediction: match: %w", shared.ErrNotFound)
	}

	poolID := shared.PoolID(input.PoolID)
	userID := shared.UserID(input.UserID)

	// Verify user is a member of the pool
	members, err := uc.pools.FindMembers(ctx, poolID)
	if err != nil {
		return nil, fmt.Errorf("submit_prediction: pool: %w", err)
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

	// Check prediction cutoff: must be before match kickoff
	if !m.CanPredict(time.Now()) {
		return nil, fmt.Errorf("%w: match has already started", shared.ErrInvalidInput)
	}

	// Upsert prediction
	predID := shared.PredictionID(uuid.Must(uuid.NewV7()).String())
	p, err := prediction.New(predID, userID, poolID, shared.MatchID(input.MatchID), input.HomeGoals, input.AwayGoals)
	if err != nil {
		return nil, fmt.Errorf("submit_prediction: %w", err)
	}

	if err := uc.predictions.Upsert(ctx, p); err != nil {
		return nil, fmt.Errorf("submit_prediction: upsert: %w", err)
	}

	return &SubmitPredictionOutput{
		PredictionID: string(predID),
		HomeGoals:    input.HomeGoals,
		AwayGoals:    input.AwayGoals,
	}, nil
}
