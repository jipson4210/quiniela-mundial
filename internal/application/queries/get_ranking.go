package queries

import (
	"context"
	"sort"

	"github.com/josemontalban/quiniela-mundial/internal/domain/scoring"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/user"
)

type GetRanking struct {
	scores scoring.Repository
	users  user.Repository
}

func NewGetRanking(scores scoring.Repository, users user.Repository) *GetRanking {
	return &GetRanking{scores: scores, users: users}
}

type RankingEntry struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	TotalPoints int    `json:"total_points"`
}

func (q *GetRanking) Execute(ctx context.Context, poolID string) ([]RankingEntry, error) {
	entries, err := q.scores.FindByPool(ctx, shared.PoolID(poolID))
	if err != nil {
		return nil, err
	}

	// Aggregate points by user
	pointsByUser := make(map[shared.UserID]int)
	for _, e := range entries {
		pointsByUser[e.UserID()] += e.Points()
	}

	result := make([]RankingEntry, 0, len(pointsByUser))
	for uid, pts := range pointsByUser {
		u, err := q.users.FindByID(ctx, uid)
		name := string(uid)
		if err == nil {
			name = u.DisplayName()
		}
		result = append(result, RankingEntry{
			UserID:      string(uid),
			DisplayName: name,
			TotalPoints: pts,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalPoints > result[j].TotalPoints
	})

	return result, nil
}
