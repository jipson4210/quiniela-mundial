package scoring

import (
	"context"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// SourceType classifies the origin of a score entry.
type SourceType string

const (
	SourceMatch          SourceType = "match"
	SourceBracketStage   SourceType = "bracket_stage"
	SourceBracketThird   SourceType = "bracket_third"
	SourceBracketChampion SourceType = "bracket_champion"
)

// ScoreEntry records points awarded for a prediction.
type ScoreEntry struct {
	id         shared.ScoreEntryID
	userID     shared.UserID
	poolID     shared.PoolID
	sourceType SourceType
	sourceRef  string // match_id, "stage:team_id", "third_place", "champion"
	points     int
	computedAt time.Time
	version    int
}

func NewScoreEntry(id shared.ScoreEntryID, userID shared.UserID, poolID shared.PoolID, sourceType SourceType, sourceRef string, points int) (*ScoreEntry, error) {
	if points < 0 {
		return nil, shared.ErrInvalidInput
	}
	return &ScoreEntry{
		id: id, userID: userID, poolID: poolID,
		sourceType: sourceType, sourceRef: sourceRef, points: points,
	}, nil
}

func ReconstructScoreEntry(id shared.ScoreEntryID, userID shared.UserID, poolID shared.PoolID, sourceType SourceType, sourceRef string, points int, computedAt time.Time, version int) *ScoreEntry {
	return &ScoreEntry{
		id: id, userID: userID, poolID: poolID,
		sourceType: sourceType, sourceRef: sourceRef,
		points: points, computedAt: computedAt, version: version,
	}
}

// Accessors
func (se *ScoreEntry) ID() shared.ScoreEntryID   { return se.id }
func (se *ScoreEntry) UserID() shared.UserID     { return se.userID }
func (se *ScoreEntry) PoolID() shared.PoolID      { return se.poolID }
func (se *ScoreEntry) SourceType() SourceType     { return se.sourceType }
func (se *ScoreEntry) SourceRef() string          { return se.sourceRef }
func (se *ScoreEntry) Points() int                { return se.points }
func (se *ScoreEntry) ComputedAt() time.Time      { return se.computedAt }
func (se *ScoreEntry) Version() int               { return se.version }

// Repository defines persistence for score entries.
type Repository interface {
	Upsert(ctx context.Context, se *ScoreEntry) error
	FindByUserAndPool(ctx context.Context, userID shared.UserID, poolID shared.PoolID) ([]*ScoreEntry, error)
	FindByPool(ctx context.Context, poolID shared.PoolID) ([]*ScoreEntry, error)
	SumByUserAndPool(ctx context.Context, userID shared.UserID, poolID shared.PoolID) (int, error)
	DeleteBySourceRef(ctx context.Context, sourceType SourceType, sourceRef string) error
}
