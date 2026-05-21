package tournament

import (
	"errors"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// Tournament is the aggregate root for the World Cup structure.
type Tournament struct {
	id        shared.TournamentID
	name      string
	startsAt  time.Time
	endsAt    time.Time
	groups    []Group
	stages    []StageDefinition
	teams     []shared.TeamID
	createdAt time.Time
}

// Group represents a World Cup group (A through L).
type Group struct {
	id    shared.GroupID
	name  string // "A".."L"
	teams []shared.TeamID
}

// StageDefinition configures points awarded per correct team at each stage.
type StageDefinition struct {
	stage               Stage
	pointsPerCorrectTeam int
	description          string
}

// Scoring points per stage (matches SCORING-RULES.md)
var DefaultStagePoints = map[Stage]int{
	StageRoundOf32:   3,
	StageRoundOf16:   4,
	StageQuarterFinal: 5,
	StageSemiFinal:   10,
	StageFinal:       15,
	StageThirdPlace:  20,
}

// Errors
var (
	ErrInvalidDates = errors.New("starts_at must be before ends_at")
	ErrGroupCount   = errors.New("World Cup 2026 must have exactly 12 groups")
	ErrTeamsPerGroup = errors.New("each group must have exactly 4 teams")
)

// New creates a Tournament with validation.
func New(id shared.TournamentID, name string, startsAt, endsAt time.Time) (*Tournament, error) {
	if !startsAt.Before(endsAt) {
		return nil, ErrInvalidDates
	}
	return &Tournament{
		id:       id,
		name:     name,
		startsAt: startsAt,
		endsAt:   endsAt,
		groups:   make([]Group, 0, 12),
		stages:   make([]StageDefinition, 0, len(AllStages)),
		teams:    make([]shared.TeamID, 0, 48),
	}, nil
}

// Reconstruct hydrates a Tournament from persistence without re-validating.
func Reconstruct(
	id shared.TournamentID, name string,
	startsAt, endsAt, createdAt time.Time,
	groups []Group, stages []StageDefinition,
	teams []shared.TeamID,
) *Tournament {
	return &Tournament{
		id:        id,
		name:      name,
		startsAt:  startsAt,
		endsAt:    endsAt,
		createdAt: createdAt,
		groups:    groups,
		stages:    stages,
		teams:     teams,
	}
}

// Accessors
func (t *Tournament) ID() shared.TournamentID { return t.id }
func (t *Tournament) Name() string             { return t.name }
func (t *Tournament) StartsAt() time.Time      { return t.startsAt }
func (t *Tournament) EndsAt() time.Time        { return t.endsAt }
func (t *Tournament) Groups() []Group          { return t.groups }
func (t *Tournament) Stages() []StageDefinition { return t.stages }
func (t *Tournament) Teams() []shared.TeamID   { return t.teams }
func (t *Tournament) CreatedAt() time.Time     { return t.createdAt }
func (t *Tournament) TeamCount() int           { return len(t.teams) }

// CanPredictBracket returns true if the tournament hasn't started yet.
func (t *Tournament) CanPredictBracket(at time.Time) bool {
	return at.Before(t.startsAt)
}

// AddGroup adds a World Cup group after validating it has exactly 4 teams.
func (t *Tournament) AddGroup(g Group) error {
	if len(g.teams) != 4 {
		return ErrTeamsPerGroup
	}
	t.groups = append(t.groups, g)
	for _, teamID := range g.teams {
		t.teams = append(t.teams, teamID)
	}
	return nil
}

// AddStage adds a stage definition.
func (t *Tournament) AddStage(sd StageDefinition) {
	t.stages = append(t.stages, sd)
}

// ValidateComplete checks the tournament meets World Cup 2026 requirements.
func (t *Tournament) ValidateComplete() error {
	if len(t.groups) != 12 {
		return ErrGroupCount
	}
	if len(t.teams) != 48 {
		return errors.New("tournament must have exactly 48 teams")
	}
	return nil
}

// Group constructors / accessors
func NewGroup(id shared.GroupID, name string, teams []shared.TeamID) Group {
	return Group{id: id, name: name, teams: teams}
}
func (g Group) ID() shared.GroupID   { return g.id }
func (g Group) Name() string         { return g.name }
func (g Group) Teams() []shared.TeamID { return g.teams }

// StageDefinition constructors / accessors
func NewStageDef(s Stage, points int, desc string) StageDefinition {
	return StageDefinition{stage: s, pointsPerCorrectTeam: points, description: desc}
}
func (sd StageDefinition) Stage() Stage          { return sd.stage }
func (sd StageDefinition) Points() int            { return sd.pointsPerCorrectTeam }
func (sd StageDefinition) Description() string    { return sd.description }
