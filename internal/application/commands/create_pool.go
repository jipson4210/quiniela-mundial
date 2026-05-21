package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// CreatePoolInput holds data to create a pool.
type CreatePoolInput struct {
	Name         string
	Description  string
	CreatorID    string
	TournamentID string
}

// CreatePoolOutput holds the created pool data.
type CreatePoolOutput struct {
	PoolID string
	Name   string
}

// CreatePool creates a new private pool and adds the creator as a member.
type CreatePool struct {
	pools pool.Repository
}

func NewCreatePool(pools pool.Repository) *CreatePool {
	return &CreatePool{pools: pools}
}

func (uc *CreatePool) Execute(ctx context.Context, input CreatePoolInput) (*CreatePoolOutput, error) {
	p, err := pool.New(
		shared.PoolID(uuid.Must(uuid.NewV7()).String()),
		input.Name,
		input.Description,
		shared.UserID(input.CreatorID),
		shared.TournamentID(input.TournamentID),
	)
	if err != nil {
		return nil, fmt.Errorf("create_pool: %w", err)
	}

	if err := uc.pools.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("create_pool: save: %w", err)
	}

	// Creator joins as the first member with creator role
	pm := pool.NewPoolMember(p.ID(), p.CreatorID(), pool.RoleCreator, nil)
	if err := uc.pools.AddMember(ctx, pm); err != nil {
		return nil, fmt.Errorf("create_pool: add creator member: %w", err)
	}

	return &CreatePoolOutput{
		PoolID: string(p.ID()),
		Name:   p.Name(),
	}, nil
}
