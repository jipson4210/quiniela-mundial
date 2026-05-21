package prediction

import (
	"context"
	"errors"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// BracketPrediction is the aggregate root for a user's knockout-stage forecast.
// Coherence invariant: champion ∈ finalists, finalists ⊂ semi_finalists, etc.
type BracketPrediction struct {
	id                   shared.BracketPredID
	userID               shared.UserID
	poolID               shared.PoolID
	tournamentID         shared.TournamentID
	teamsToRoundOf32     []shared.TeamID // 32 teams
	teamsToRoundOf16     []shared.TeamID // 16 teams
	teamsToQuarterFinal  []shared.TeamID // 8 teams
	teamsToSemiFinal     []shared.TeamID // 4 teams
	teamsToFinal         []shared.TeamID // 2 teams
	thirdPlaceWinner     shared.TeamID
	champion             shared.TeamID
	submittedAt          time.Time
	updatedAt            time.Time
}

var (
	ErrInvalidTeamCount   = errors.New("invalid team count for stage")
	ErrChampionNotInFinal = errors.New("champion must be one of the finalists")
	ErrThirdNotInSemi     = errors.New("third place winner must be a semi-finalist")
	ErrThirdInFinal       = errors.New("third place winner cannot be a finalist")
	ErrSubsetViolation    = errors.New("bracket coherence violation: later stage must be subset of earlier stage")
)

// NewBracket creates a BracketPrediction with full coherence validation.
func NewBracket(
	id shared.BracketPredID,
	userID shared.UserID,
	poolID shared.PoolID,
	tournamentID shared.TournamentID,
	r32, r16, qf, sf, f []shared.TeamID,
	thirdPlace shared.TeamID,
	champion shared.TeamID,
) (*BracketPrediction, error) {
	bp := &BracketPrediction{
		id:               id,
		userID:           userID,
		poolID:           poolID,
		tournamentID:     tournamentID,
		teamsToRoundOf32: r32,
		teamsToRoundOf16: r16,
		teamsToQuarterFinal: qf,
		teamsToSemiFinal: sf,
		teamsToFinal:     f,
		thirdPlaceWinner: thirdPlace,
		champion:         champion,
	}
	if err := bp.validate(); err != nil {
		return nil, err
	}
	return bp, nil
}

func (bp *BracketPrediction) validate() error {
	// Count checks
	if len(bp.teamsToRoundOf32) != 32 {
		return ErrInvalidTeamCount
	}
	if len(bp.teamsToRoundOf16) != 16 {
		return ErrInvalidTeamCount
	}
	if len(bp.teamsToQuarterFinal) != 8 {
		return ErrInvalidTeamCount
	}
	if len(bp.teamsToSemiFinal) != 4 {
		return ErrInvalidTeamCount
	}
	if len(bp.teamsToFinal) != 2 {
		return ErrInvalidTeamCount
	}

	// Subset checks: each stage must be a subset of the previous
	if !isSubset(bp.teamsToRoundOf16, bp.teamsToRoundOf32) {
		return ErrSubsetViolation
	}
	if !isSubset(bp.teamsToQuarterFinal, bp.teamsToRoundOf16) {
		return ErrSubsetViolation
	}
	if !isSubset(bp.teamsToSemiFinal, bp.teamsToQuarterFinal) {
		return ErrSubsetViolation
	}
	if !isSubset(bp.teamsToFinal, bp.teamsToSemiFinal) {
		return ErrSubsetViolation
	}

	// Champion must be a finalist
	if !contains(bp.teamsToFinal, bp.champion) {
		return ErrChampionNotInFinal
	}

	// Third place must be a semi-finalist
	if !contains(bp.teamsToSemiFinal, bp.thirdPlaceWinner) {
		return ErrThirdNotInSemi
	}

	// Third place cannot be a finalist
	if contains(bp.teamsToFinal, bp.thirdPlaceWinner) {
		return ErrThirdInFinal
	}

	return nil
}

func isSubset(subset, superset []shared.TeamID) bool {
	set := make(map[shared.TeamID]struct{}, len(superset))
	for _, t := range superset {
		set[t] = struct{}{}
	}
	for _, t := range subset {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}

func contains(slice []shared.TeamID, id shared.TeamID) bool {
	for _, t := range slice {
		if t == id {
			return true
		}
	}
	return false
}

// Reconstruct hydrates a BracketPrediction from persistence without re-validating.
func ReconstructBracket(
	id shared.BracketPredID,
	userID shared.UserID, poolID shared.PoolID, tournamentID shared.TournamentID,
	r32, r16, qf, sf, f []shared.TeamID,
	thirdPlace, champion shared.TeamID,
	submittedAt, updatedAt time.Time,
) *BracketPrediction {
	return &BracketPrediction{
		id: id, userID: userID, poolID: poolID, tournamentID: tournamentID,
		teamsToRoundOf32: r32, teamsToRoundOf16: r16,
		teamsToQuarterFinal: qf, teamsToSemiFinal: sf,
		teamsToFinal: f, thirdPlaceWinner: thirdPlace, champion: champion,
		submittedAt: submittedAt, updatedAt: updatedAt,
	}
}

// Accessors
func (bp *BracketPrediction) ID() shared.BracketPredID          { return bp.id }
func (bp *BracketPrediction) UserID() shared.UserID              { return bp.userID }
func (bp *BracketPrediction) PoolID() shared.PoolID               { return bp.poolID }
func (bp *BracketPrediction) TournamentID() shared.TournamentID   { return bp.tournamentID }
func (bp *BracketPrediction) TeamsToRoundOf32() []shared.TeamID   { return bp.teamsToRoundOf32 }
func (bp *BracketPrediction) TeamsToRoundOf16() []shared.TeamID   { return bp.teamsToRoundOf16 }
func (bp *BracketPrediction) TeamsToQuarterFinal() []shared.TeamID { return bp.teamsToQuarterFinal }
func (bp *BracketPrediction) TeamsToSemiFinal() []shared.TeamID   { return bp.teamsToSemiFinal }
func (bp *BracketPrediction) TeamsToFinal() []shared.TeamID       { return bp.teamsToFinal }
func (bp *BracketPrediction) ThirdPlaceWinner() shared.TeamID    { return bp.thirdPlaceWinner }
func (bp *BracketPrediction) Champion() shared.TeamID            { return bp.champion }
func (bp *BracketPrediction) SubmittedAt() time.Time             { return bp.submittedAt }
func (bp *BracketPrediction) UpdatedAt() time.Time               { return bp.updatedAt }

// BracketRepository defines persistence for bracket predictions.
type BracketRepository interface {
	Upsert(ctx context.Context, bp *BracketPrediction) error
	FindByUserAndPool(ctx context.Context, userID shared.UserID, poolID shared.PoolID) (*BracketPrediction, error)
}
