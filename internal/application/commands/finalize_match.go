package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// FinalizeMatchInput holds official match result data.
type FinalizeMatchInput struct {
	MatchID               string
	HomeGoals             int
	AwayGoals             int
	HomeGoalsET           *int
	AwayGoalsET           *int
	HomeGoalsPenalties    *int
	AwayGoalsPenalties    *int
	Source                string // "manual", "api_footballdata", etc.
}

// FinalizeMatchOutput holds the result of finalization.
type FinalizeMatchOutput struct {
	MatchID    string
	Status     string
	FinalizedAt string
}

// FinalizeMatch applies an official result to a match, transitioning it to finished.
type FinalizeMatch struct {
	matches match.Repository
}

func NewFinalizeMatch(matches match.Repository) *FinalizeMatch {
	return &FinalizeMatch{matches: matches}
}

func (uc *FinalizeMatch) Execute(ctx context.Context, input FinalizeMatchInput) (*FinalizeMatchOutput, error) {
	m, err := uc.matches.FindByID(ctx, shared.MatchID(input.MatchID))
	if err != nil {
		return nil, fmt.Errorf("finalize: match: %w", shared.ErrNotFound)
	}

	now := time.Now()
	extResult := match.ExternalResult{
		HomeGoals:               &input.HomeGoals,
		AwayGoals:               &input.AwayGoals,
		HomeGoalsAfterET:        input.HomeGoalsET,
		AwayGoalsAfterET:        input.AwayGoalsET,
		HomeGoalsAfterPenalties: input.HomeGoalsPenalties,
		AwayGoalsAfterPenalties: input.AwayGoalsPenalties,
		Source:                  input.Source,
		FetchedAt:               now,
	}

	if err := m.FinalizeWith(extResult); err != nil {
		return nil, fmt.Errorf("finalize: %w", err)
	}

	// Persist
	if err := uc.matches.UpdateResult(ctx, m); err != nil {
		return nil, fmt.Errorf("finalize: persist result: %w", err)
	}

	return &FinalizeMatchOutput{
		MatchID:    string(m.ID()),
		Status:     string(m.Status()),
		FinalizedAt: now.Format(time.RFC3339),
	}, nil
}
