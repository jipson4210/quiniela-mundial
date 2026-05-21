package match

import (
	"context"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
)

// Status represents the lifecycle of a match.
type Status string

const (
	StatusScheduled  Status = "scheduled"
	StatusInProgress Status = "in_progress"
	StatusFinished   Status = "finished"
	StatusCancelled  Status = "cancelled"
)

// ResultSource records where the result came from.
type ResultSource string

const (
	SourceFootballdata ResultSource = "api_footballdata"
	SourceBalldontlie   ResultSource = "api_balldontlie"
	SourceManual        ResultSource = "manual"
)

// Match is the aggregate root for a tournament match.
type Match struct {
	id            shared.MatchID
	tournamentID  shared.TournamentID
	stage         tournament.Stage
	groupID       *shared.GroupID // nil for knockout stages
	homeTeamID    shared.TeamID
	awayTeamID    shared.TeamID
	kickoffAt     time.Time
	venue         string
	status        Status
	result        *Result
	createdAt     time.Time
}

// Result is a value object holding the final score.
type Result struct {
	homeGoals            int
	awayGoals            int
	homeGoalsET          *int
	awayGoalsET          *int
	homeGoalsPenalties   *int
	awayGoalsPenalties   *int
	source               ResultSource
	finalizedAt          time.Time
}

// New creates a scheduled Match.
func New(
	id shared.MatchID,
	tournamentID shared.TournamentID,
	stage tournament.Stage,
	groupID *shared.GroupID,
	homeTeamID, awayTeamID shared.TeamID,
	kickoffAt time.Time,
	venue string,
) (*Match, error) {
	if homeTeamID == awayTeamID {
		return nil, shared.ErrInvalidInput
	}
	return &Match{
		id:           id,
		tournamentID: tournamentID,
		stage:        stage,
		groupID:      groupID,
		homeTeamID:   homeTeamID,
		awayTeamID:   awayTeamID,
		kickoffAt:    kickoffAt,
		venue:        venue,
		status:       StatusScheduled,
	}, nil
}

// Reconstruct hydrates a Match from persistence.
func Reconstruct(
	id shared.MatchID, tournamentID shared.TournamentID,
	stage tournament.Stage, groupID *shared.GroupID,
	homeTeamID, awayTeamID shared.TeamID,
	kickoffAt time.Time, venue string,
	status Status, result *Result, createdAt time.Time,
) *Match {
	return &Match{
		id: id, tournamentID: tournamentID,
		stage: stage, groupID: groupID,
		homeTeamID: homeTeamID, awayTeamID: awayTeamID,
		kickoffAt: kickoffAt, venue: venue,
		status: status, result: result, createdAt: createdAt,
	}
}

// Accessors
func (m *Match) ID() shared.MatchID            { return m.id }
func (m *Match) TournamentID() shared.TournamentID { return m.tournamentID }
func (m *Match) Stage() tournament.Stage        { return m.stage }
func (m *Match) GroupID() *shared.GroupID       { return m.groupID }
func (m *Match) HomeTeamID() shared.TeamID      { return m.homeTeamID }
func (m *Match) AwayTeamID() shared.TeamID      { return m.awayTeamID }
func (m *Match) KickoffAt() time.Time           { return m.kickoffAt }
func (m *Match) Venue() string                  { return m.venue }
func (m *Match) Status() Status                 { return m.status }
func (m *Match) Result() *Result                { return m.result }

// CanPredict returns true if predictions are still open for this match.
func (m *Match) CanPredict(at time.Time) bool {
	return at.Before(m.kickoffAt)
}

// FinalizeWith applies an external result to finalize the match.
func (m *Match) FinalizeWith(ext ExternalResult) error {
	if m.status == StatusFinished {
		return nil // idempotent
	}
	m.status = StatusFinished
	m.result = &Result{
		homeGoals:          ptrVal(ext.HomeGoals),
		awayGoals:          ptrVal(ext.AwayGoals),
		homeGoalsET:        ext.HomeGoalsAfterET,
		awayGoalsET:        ext.AwayGoalsAfterET,
		homeGoalsPenalties: ext.HomeGoalsAfterPenalties,
		awayGoalsPenalties: ext.AwayGoalsAfterPenalties,
		source:             ResultSource(ext.Source),
		finalizedAt:        ext.FetchedAt,
	}
	return nil
}

func ptrVal(p *int) int {
	if p == nil { return 0 }
	return *p
}

// Result accessors
func (r *Result) HomeGoals() int              { return r.homeGoals }
func (r *Result) AwayGoals() int              { return r.awayGoals }
func (r *Result) HomeGoalsET() *int           { return r.homeGoalsET }
func (r *Result) AwayGoalsET() *int           { return r.awayGoalsET }
func (r *Result) HomeGoalsPen() *int          { return r.homeGoalsPenalties }
func (r *Result) AwayGoalsPen() *int          { return r.awayGoalsPenalties }
func (r *Result) Source() ResultSource        { return r.source }
func (r *Result) FinalizedAt() time.Time      { return r.finalizedAt }

// ReconstructResult hydrates a Result from persistence.
func ReconstructResult(homeGoals, awayGoals int, homeEt, awayEt, homePen, awayPen *int, source ResultSource, finalizedAt time.Time) *Result {
	return &Result{
		homeGoals:          homeGoals,
		awayGoals:          awayGoals,
		homeGoalsET:        homeEt,
		awayGoalsET:        awayEt,
		homeGoalsPenalties: homePen,
		awayGoalsPenalties: awayPen,
		source:             source,
		finalizedAt:        finalizedAt,
	}
}

// ExternalResult is the domain port — adapters translate their DTOs into this.
type ExternalResult struct {
	ExternalMatchID         string
	HomeTeamCode            string
	AwayTeamCode            string
	KickoffAt               time.Time
	Status                  string
	HomeGoals               *int
	AwayGoals               *int
	HomeGoalsAfterET        *int
	AwayGoalsAfterET        *int
	HomeGoalsAfterPenalties *int
	AwayGoalsAfterPenalties *int
	Source                  string
	FetchedAt               time.Time
}

// ResultProvider is the port implemented by external API adapters.
type ResultProvider interface {
	FetchResults(ctx context.Context, from, to time.Time) ([]ExternalResult, error)
}

// Repository defines persistence for matches.
type Repository interface {
	Save(ctx context.Context, m *Match) error
	SaveBatch(ctx context.Context, matches []*Match) error
	FindByID(ctx context.Context, id shared.MatchID) (*Match, error)
	FindByTournament(ctx context.Context, tournamentID shared.TournamentID) ([]*Match, error)
	FindByStage(ctx context.Context, tournamentID shared.TournamentID, stage tournament.Stage) ([]*Match, error)
	FindByTeamAndKickoff(ctx context.Context, homeTeamID, awayTeamID shared.TeamID, kickoffAt time.Time) (*Match, error)
	CreateStageGroup(ctx context.Context, tournamentID shared.TournamentID, groupID shared.GroupID, name string) error
	AddTeamToGroup(ctx context.Context, groupID shared.GroupID, teamID shared.TeamID) error
	UpdateResult(ctx context.Context, m *Match) error
}
