package queries

import (
	"context"

	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

type GetUserPools struct {
	pools pool.Repository
}

func NewGetUserPools(pools pool.Repository) *GetUserPools {
	return &GetUserPools{pools: pools}
}

type PoolDTO struct {
	ID          string `json:"PoolID"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
}

func (q *GetUserPools) Execute(ctx context.Context, userID string) ([]PoolDTO, error) {
	pools, err := q.pools.FindByUser(ctx, shared.UserID(userID))
	if err != nil {
		return nil, err
	}
	result := make([]PoolDTO, 0, len(pools))
	for _, p := range pools {
		result = append(result, PoolDTO{
			ID:          string(p.ID()),
			Name:        p.Name(),
			Description: p.Description(),
		})
	}
	return result, nil
}
