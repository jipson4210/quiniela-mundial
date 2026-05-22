// Package prediction implements the MatchPrediction entity for the Quiniela bounded context.
package prediction

import (
	"context"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// MatchPrediction represents a user's goals prediction for a specific match in a pool.
type MatchPrediction struct {
	id          shared.PredictionID
	userID      shared.UserID
	poolID      shared.PoolID
	matchID     shared.MatchID
	homeGoals   int
	awayGoals   int
	submittedAt time.Time
	updatedAt   time.Time
}

// New creates a MatchPrediction with validation.
func New(id shared.PredictionID, userID shared.UserID, poolID shared.PoolID, matchID shared.MatchID, homeGoals, awayGoals int) (*MatchPrediction, error) {
	if homeGoals < 0 || homeGoals > 30 || awayGoals < 0 || awayGoals > 30 {
		return nil, shared.ErrInvalidInput
	}
	return &MatchPrediction{
		id:        id,
		userID:    userID,
		poolID:    poolID,
		matchID:   matchID,
		homeGoals: homeGoals,
		awayGoals: awayGoals,
	}, nil
}

// Reconstruct hydrates a MatchPrediction from persistence.
func Reconstruct(id shared.PredictionID, userID shared.UserID, poolID shared.PoolID, matchID shared.MatchID, homeGoals, awayGoals int, submittedAt, updatedAt time.Time) *MatchPrediction {
	return &MatchPrediction{
		id: id, userID: userID, poolID: poolID, matchID: matchID,
		homeGoals: homeGoals, awayGoals: awayGoals,
		submittedAt: submittedAt, updatedAt: updatedAt,
	}
}

// Accessors
func (p *MatchPrediction) ID() shared.PredictionID  { return p.id }
func (p *MatchPrediction) UserID() shared.UserID     { return p.userID }
func (p *MatchPrediction) PoolID() shared.PoolID      { return p.poolID }
func (p *MatchPrediction) MatchID() shared.MatchID    { return p.matchID }
func (p *MatchPrediction) HomeGoals() int             { return p.homeGoals }
func (p *MatchPrediction) AwayGoals() int             { return p.awayGoals }
func (p *MatchPrediction) SubmittedAt() time.Time     { return p.submittedAt }
func (p *MatchPrediction) UpdatedAt() time.Time       { return p.updatedAt }

// Repository defines persistence for match predictions.
type Repository interface {
	Upsert(ctx context.Context, p *MatchPrediction) error
	FindByUserAndMatch(ctx context.Context, userID shared.UserID, poolID shared.PoolID, matchID shared.MatchID) (*MatchPrediction, error)
	FindByPoolAndMatch(ctx context.Context, poolID shared.PoolID, matchID shared.MatchID) ([]*MatchPrediction, error)
	FindByUserAndPool(ctx context.Context, userID shared.UserID, poolID shared.PoolID) ([]*MatchPrediction, error)
	FindDistinctPoolsByMatch(ctx context.Context, matchID shared.MatchID) ([]shared.PoolID, error)
}
