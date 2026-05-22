package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/scoring"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
)

// BracketStageInput defines which stage to compute and the actual advancing teams.
type BracketStageInput struct {
	PoolID      string
	Stage       tournament.Stage       // round_of_32, round_of_16, quarter_final, semi_final, final, third_place
	ActualTeams []string               // team IDs that actually advanced to/reached this stage
	Champion    string                 // for "champion" stage
	ThirdPlace  string                 // for "third_place" stage
}

// BracketStageOutput holds the result of bracket scoring for a stage.
type BracketStageOutput struct {
	Stage            string
	PredictionsFound int
	PointsAwarded    int
	UsersScored      int
}

// ComputeBracketPoints scores bracket predictions against actual advancing teams.
type ComputeBracketPoints struct {
	brackets prediction.BracketRepository
	pools    pool.Repository
	scores   scoring.Repository
}

func NewComputeBracketPoints(brackets prediction.BracketRepository, pools pool.Repository, scores scoring.Repository) *ComputeBracketPoints {
	return &ComputeBracketPoints{brackets: brackets, pools: pools, scores: scores}
}

// Execute computes bracket points for a specific stage across all members of a pool.
func (uc *ComputeBracketPoints) Execute(ctx context.Context, input BracketStageInput) (*BracketStageOutput, error) {
	poolID := shared.PoolID(input.PoolID)

	members, err := uc.pools.FindMembers(ctx, poolID)
	if err != nil {
		return nil, fmt.Errorf("bracket_points: pool: %w", err)
	}

	output := &BracketStageOutput{Stage: string(input.Stage)}
	totalPoints := 0
	usersScored := 0

	for _, member := range members {
		bp, err := uc.brackets.FindByUserAndPool(ctx, member.UserID(), poolID)
		if err != nil {
			continue // no bracket prediction for this user
		}
		output.PredictionsFound++

		pts := uc.computeStagePoints(bp, input)
		if pts == 0 {
			continue
		}

		// Create score entry
		sourceType := scoring.SourceBracketStage
		sourceRef := fmt.Sprintf("%s:%s", input.Stage, member.UserID())
		if input.Stage == "champion" {
			sourceType = scoring.SourceBracketChampion
			sourceRef = "champion"
		} else if input.Stage == "third_place" {
			sourceType = scoring.SourceBracketThird
			sourceRef = "third_place"
		}

		entry, err := scoring.NewScoreEntry(
			shared.ScoreEntryID(uuid.Must(uuid.NewV7()).String()),
			member.UserID(), poolID, sourceType, sourceRef, pts,
		)
		if err != nil {
			return nil, fmt.Errorf("bracket_points: create entry: %w", err)
		}

		if err := uc.scores.Upsert(ctx, entry); err != nil {
			return nil, fmt.Errorf("bracket_points: upsert: %w", err)
		}

		totalPoints += pts
		usersScored++
	}

	output.PointsAwarded = totalPoints
	output.UsersScored = usersScored

	log.Printf("[bracket] stage=%s pool=%s: %d users, %d total pts",
		input.Stage, input.PoolID, usersScored, totalPoints)

	return output, nil
}

func (uc *ComputeBracketPoints) computeStagePoints(bp *prediction.BracketPrediction, input BracketStageInput) int {
	actualSet := make(map[shared.TeamID]bool, len(input.ActualTeams))
	for _, t := range input.ActualTeams {
		actualSet[shared.TeamID(t)] = true
	}

	switch input.Stage {
	case tournament.StageRoundOf32:
		return countMatches(bp.TeamsToRoundOf32(), actualSet) * 3
	case tournament.StageRoundOf16:
		return countMatches(bp.TeamsToRoundOf16(), actualSet) * 4
	case tournament.StageQuarterFinal:
		return countMatches(bp.TeamsToQuarterFinal(), actualSet) * 5
	case tournament.StageSemiFinal:
		return countMatches(bp.TeamsToSemiFinal(), actualSet) * 10
	case tournament.StageFinal:
		return countMatches(bp.TeamsToFinal(), actualSet) * 0 // final teams earn points via "reached the final" count
	case tournament.StageThirdPlace:
		if shared.TeamID(input.ThirdPlace) == bp.ThirdPlaceWinner() {
			return 15
		}
		return 0
	case "champion":
		if shared.TeamID(input.Champion) == bp.Champion() {
			return 20
		}
		return 0
	default:
		return 0
	}
}

func countMatches(predicted []shared.TeamID, actual map[shared.TeamID]bool) int {
	count := 0
	for _, t := range predicted {
		if actual[t] {
			count++
		}
	}
	return count
}
