package team

import (
	"context"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// Team represents a national team participating in the World Cup.
type Team struct {
	id            shared.TeamID
	code          string // FIFA 3-letter code: "ARG", "BRA"
	name          string
	flagURL       string
	confederation string
	tournamentID  shared.TournamentID
}

// New creates a new Team.
func New(id shared.TeamID, code, name, flagURL, confederation string, tournamentID shared.TournamentID) (*Team, error) {
	if len(code) < 3 || len(code) > 4 {
		return nil, shared.ErrInvalidInput
	}
	if name == "" {
		return nil, shared.ErrInvalidInput
	}
	return &Team{
		id:            id,
		code:          code,
		name:          name,
		flagURL:       flagURL,
		confederation: confederation,
		tournamentID:  tournamentID,
	}, nil
}

// Reconstruct hydrates a Team from persistence.
func Reconstruct(id shared.TeamID, code, name, flagURL, confederation string, tournamentID shared.TournamentID) *Team {
	return &Team{
		id:            id,
		code:          code,
		name:          name,
		flagURL:       flagURL,
		confederation: confederation,
		tournamentID:  tournamentID,
	}
}

// Accessors
func (t *Team) ID() shared.TeamID          { return t.id }
func (t *Team) Code() string               { return t.code }
func (t *Team) Name() string               { return t.name }
func (t *Team) FlagURL() string            { return t.flagURL }
func (t *Team) Confederation() string      { return t.confederation }
func (t *Team) TournamentID() shared.TournamentID { return t.tournamentID }

// ExternalID maps a team to its identifier in an external data source.
type ExternalID struct {
	teamID     shared.TeamID
	source     string // "openfootball", "footballdata", "balldontlie"
	externalID string
}

func NewExternalID(teamID shared.TeamID, source, externalID string) ExternalID {
	return ExternalID{teamID: teamID, source: source, externalID: externalID}
}
func (e ExternalID) TeamID() shared.TeamID { return e.teamID }
func (e ExternalID) Source() string        { return e.source }
func (e ExternalID) ExternalID() string    { return e.externalID }

// Repository defines persistence for teams.
type Repository interface {
	Save(ctx context.Context, t *Team) error
	SaveBatch(ctx context.Context, teams []*Team) error
	FindByID(ctx context.Context, id shared.TeamID) (*Team, error)
	FindByCode(ctx context.Context, code string, tournamentID shared.TournamentID) (*Team, error)
	FindByTournament(ctx context.Context, tournamentID shared.TournamentID) ([]*Team, error)
	SaveExternalID(ctx context.Context, e ExternalID) error
	FindByExternalID(ctx context.Context, source, externalID string) (*Team, error)
}
