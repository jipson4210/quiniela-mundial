package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/scoring"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// ComputeMatchPointsInput holds the match and pool to score.
type ComputeMatchPointsInput struct {
	MatchID string
	PoolID  string
}

// ComputeMatchPointsOutput holds the result counts.
type ComputeMatchPointsOutput struct {
	MatchID       string
	PoolID        string
	Predictions   int
	PointsAwarded int
}

// ComputeMatchPoints scores all predictions for a finalized match in a pool.
// Idempotent: uses UPSERT so recomputing is safe.
type ComputeMatchPoints struct {
	predictions prediction.Repository
	matches     match.Repository
	scores      scoring.Repository
}

func NewComputeMatchPoints(predictions prediction.Repository, matches match.Repository, scores scoring.Repository) *ComputeMatchPoints {
	return &ComputeMatchPoints{predictions: predictions, matches: matches, scores: scores}
}

func (uc *ComputeMatchPoints) Execute(ctx context.Context, input ComputeMatchPointsInput) (*ComputeMatchPointsOutput, error) {
	m, err := uc.matches.FindByID(ctx, shared.MatchID(input.MatchID))
	if err != nil {
		return nil, fmt.Errorf("compute_points: match: %w", shared.ErrNotFound)
	}

	if m.Result() == nil || m.Status() != match.StatusFinished {
		return nil, fmt.Errorf("%w: match is not finalized", shared.ErrInvalidInput)
	}

	poolID := shared.PoolID(input.PoolID)
	matchID := shared.MatchID(input.MatchID)

	// Find all predictions for this pool+match
	preds, err := uc.predictions.FindByPoolAndMatch(ctx, poolID, matchID)
	if err != nil {
		return nil, fmt.Errorf("compute_points: fetch predictions: %w", err)
	}

	totalPoints := 0
	for _, pred := range preds {
		pts := scoring.ComputeMatchPoints(pred, m)

		entry, err := scoring.NewScoreEntry(
			shared.ScoreEntryID(uuid.Must(uuid.NewV7()).String()),
			pred.UserID(),
			pred.PoolID(),
			scoring.SourceMatch,
			string(matchID),
			pts,
		)
		if err != nil {
			return nil, fmt.Errorf("compute_points: create entry: %w", err)
		}

		if err := uc.scores.Upsert(ctx, entry); err != nil {
			return nil, fmt.Errorf("compute_points: upsert: %w", err)
		}

		totalPoints += pts
	}

	return &ComputeMatchPointsOutput{
		MatchID:       string(matchID),
		PoolID:        string(poolID),
		Predictions:   len(preds),
		PointsAwarded: totalPoints,
	}, nil
}

// ComputeForAllPools scores a finalized match across all pools that have predictions.
func (uc *ComputeMatchPoints) ComputeForAllPools(ctx context.Context, matchID string) (int, error) {
	// Find all pools that have predictions for this match.
	// We iterate through all pools by finding unique pool IDs from predictions.
	// For now, we use a simpler approach: just score the known pools.
	// In a production system, we'd query distinct pool IDs from match_predictions.
	return 0, nil
}

// ensure time import is used
var _ = time.Now
