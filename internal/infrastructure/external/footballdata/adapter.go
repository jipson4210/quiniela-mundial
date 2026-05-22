package footballdata

import (
	"context"
	"fmt"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
)

// Adapter implements match.ResultProvider using football-data.org API.
type Adapter struct {
	client *Client
}

func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

// FetchResults downloads finished matches from football-data.org and maps them to domain ExternalResult.
func (a *Adapter) FetchResults(ctx context.Context, from, to time.Time) ([]match.ExternalResult, error) {
	dto, err := a.client.FetchMatches(ctx, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("footballdata adapter: %w", err)
	}

	var results []match.ExternalResult
	for _, m := range dto.Matches {
		// Only process finished matches with scores
		if m.Status != "FINISHED" || m.Score.FullTime.Home == nil {
			continue
		}

		homeGoals := *m.Score.FullTime.Home
		awayGoals := *m.Score.FullTime.Away

		var homeET, awayET *int
		if m.Score.ExtraTime != nil {
			homeET = m.Score.ExtraTime.Home
			awayET = m.Score.ExtraTime.Away
		}

		var homePen, awayPen *int
		if m.Score.Penalties != nil {
			homePen = m.Score.Penalties.Home
			awayPen = m.Score.Penalties.Away
		}

		results = append(results, match.ExternalResult{
			ExternalMatchID:         fmt.Sprintf("%d", m.ID),
			HomeTeamCode:            m.HomeTeam.TLA,
			AwayTeamCode:            m.AwayTeam.TLA,
			KickoffAt:               m.UTCDate,
			Status:                  "finished",
			HomeGoals:               &homeGoals,
			AwayGoals:               &awayGoals,
			HomeGoalsAfterET:        homeET,
			AwayGoalsAfterET:        awayET,
			HomeGoalsAfterPenalties: homePen,
			AwayGoalsAfterPenalties: awayPen,
			Source:                  "footballdata",
			FetchedAt:               time.Now(),
		})
	}

	return results, nil
}
