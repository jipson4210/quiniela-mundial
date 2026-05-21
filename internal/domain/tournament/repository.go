package tournament

import (
	"context"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// Repository defines persistence operations for Tournament aggregates.
type Repository interface {
	Save(ctx context.Context, t *Tournament) error
	FindByID(ctx context.Context, id shared.TournamentID) (*Tournament, error)
	FindCurrent(ctx context.Context) (*Tournament, error) // the active World Cup
}
