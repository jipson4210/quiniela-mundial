package scoring

import (
	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
)

// Winner represents the direction of a match result.
type Winner string

const (
	WinnerHome Winner = "HOME"
	WinnerAway Winner = "AWAY"
	WinnerDraw Winner = "DRAW"
)

// Strategy computes points for a specific type of prediction.
type Strategy interface {
	SourceType() string // "match", "bracket_stage", etc.
	Compute(pred interface{}, actual interface{}) int
}

// MatchScoring computes points for a match prediction against the official result.
type MatchScoring struct{}

func (s *MatchScoring) SourceType() string { return "match" }

// ComputeMatchPoints calculates points for a match prediction.
// Returns points (0-5) based on direction match (+3) and exact goals (+1 each).
func ComputeMatchPoints(pred *prediction.MatchPrediction, m *match.Match) int {
	if m.Result() == nil {
		return 0 // match not finalized
	}

	result := m.Result()
	points := 0

	// Goal accuracy: compare against regular time goals
	if pred.HomeGoals() == result.HomeGoals() {
		points++
	}
	if pred.AwayGoals() == result.AwayGoals() {
		points++
	}

	// Direction accuracy: compare against official winner
	predWinner := winnerOf(pred.HomeGoals(), pred.AwayGoals())
	actualWinner := officialWinner(m)

	if predWinner == actualWinner {
		points += 3
	}

	return points
}

// winnerOf determines the direction of a scoreline.
func winnerOf(home, away int) Winner {
	switch {
	case home > away:
		return WinnerHome
	case home < away:
		return WinnerAway
	default:
		return WinnerDraw
	}
}

// officialWinner determines the match winner considering knockout rules.
// In knockout stages, the official result includes extra time and penalties.
// In group stage, regular time goals determine the winner.
func officialWinner(m *match.Match) Winner {
	r := m.Result()
	if r == nil {
		return WinnerDraw
	}

	// Determine official goals: use post-penalties result for knockout, regular for group
	homeGoals := r.HomeGoals()
	awayGoals := r.AwayGoals()

	// Check if penalties occurred (has penalty goals set)
	if r.HomeGoalsPen() != nil && r.AwayGoalsPen() != nil {
		homePen := *r.HomeGoalsPen()
		awayPen := *r.AwayGoalsPen()
		if homePen > awayPen {
			return WinnerHome
		}
		if awayPen > homePen {
			return WinnerAway
		}
	}

	// Check extra time
	if r.HomeGoalsET() != nil && r.AwayGoalsET() != nil {
		homeET := *r.HomeGoalsET()
		awayET := *r.AwayGoalsET()
		if homeET > awayET {
			return WinnerHome
		}
		if awayET > homeET {
			return WinnerAway
		}
	}

	return winnerOf(homeGoals, awayGoals)
}
