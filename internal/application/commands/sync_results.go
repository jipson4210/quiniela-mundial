package commands

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/scoring"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/team"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
)

// SyncResults fetches external results, finalizes matches, and triggers scoring.
type SyncResults struct {
	resultProvider match.ResultProvider
	matches        match.Repository
	teams          team.Repository
	predictions    prediction.Repository
	scores         scoring.Repository
	tournaments    tournament.Repository
}

func NewSyncResults(
	resultProvider match.ResultProvider,
	matches match.Repository,
	teams team.Repository,
	predictions prediction.Repository,
	scores scoring.Repository,
	tournaments tournament.Repository,
) *SyncResults {
	return &SyncResults{
		resultProvider: resultProvider,
		matches:        matches,
		teams:          teams,
		predictions:    predictions,
		scores:         scores,
		tournaments:    tournaments,
	}
}

type SyncResult struct {
	ExternalID   string
	MatchID      string
	HomeGoals    int
	AwayGoals    int
	PoolsScored  int
	TotalPoints  int
	Skipped      bool
	Error        string
}

// Execute fetches results for a date range and syncs them.
func (uc *SyncResults) Execute(ctx context.Context, from, to time.Time) ([]SyncResult, error) {
	log.Printf("[sync] fetching results from %s to %s...", from.Format("2006-01-02"), to.Format("2006-01-02"))
	results, err := uc.resultProvider.FetchResults(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("sync: fetch: %w", err)
	}
	log.Printf("[sync] got %d finished matches from provider", len(results))

	tourn, err := uc.tournaments.FindCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("sync: tournament: %w", err)
	}

	var syncResults []SyncResult
	for _, ext := range results {
		sr := uc.syncOne(ctx, ext, tourn)
		syncResults = append(syncResults, sr)
		if sr.Error != "" {
			log.Printf("[sync] ERROR %s: %s", sr.ExternalID, sr.Error)
		} else if sr.Skipped {
			log.Printf("[sync] SKIP %s: already finalized", sr.ExternalID)
		} else {
			log.Printf("[sync] OK %s: %d-%d (%d pools, %d pts)", sr.ExternalID, sr.HomeGoals, sr.AwayGoals, sr.PoolsScored, sr.TotalPoints)
		}
	}

	return syncResults, nil
}

func (uc *SyncResults) syncOne(ctx context.Context, ext match.ExternalResult, tourn *tournament.Tournament) SyncResult {
	sr := SyncResult{ExternalID: ext.ExternalMatchID}

	// Find home/away teams by code or external ID
	homeTeam := uc.resolveTeam(ctx, ext.HomeTeamCode, tourn.ID())
	if homeTeam == nil {
		sr.Error = fmt.Sprintf("home team not found: %s", ext.HomeTeamCode)
		return sr
	}
	awayTeam := uc.resolveTeam(ctx, ext.AwayTeamCode, tourn.ID())
	if awayTeam == nil {
		sr.Error = fmt.Sprintf("away team not found: %s", ext.AwayTeamCode)
		return sr
	}

	// Find match by teams + kickoff
	m, err := uc.matches.FindByTeamAndKickoff(ctx, homeTeam.ID(), awayTeam.ID(), ext.KickoffAt)
	if err != nil {
		sr.Error = fmt.Sprintf("match not found: %s vs %s", ext.HomeTeamCode, ext.AwayTeamCode)
		return sr
	}

	if m.Status() == match.StatusFinished {
		sr.Skipped = true
		return sr
	}

	// Finalize
	if err := m.FinalizeWith(ext); err != nil {
		sr.Error = fmt.Sprintf("finalize: %s", err)
		return sr
	}
	if err := uc.matches.UpdateResult(ctx, m); err != nil {
		sr.Error = fmt.Sprintf("persist result: %s", err)
		return sr
	}

	sr.MatchID = string(m.ID())
	sr.HomeGoals = *ext.HomeGoals
	sr.AwayGoals = *ext.AwayGoals

	// Score all pools that have predictions for this match
	pools, err := uc.predictions.FindDistinctPoolsByMatch(ctx, m.ID())
	if err != nil {
		log.Printf("[sync] WARN: find pools for match %s: %v", m.ID(), err)
		return sr
	}

	for _, poolID := range pools {
		preds, err := uc.predictions.FindByPoolAndMatch(ctx, poolID, m.ID())
		if err != nil {
			continue
		}
		for _, pred := range preds {
			pts := scoring.ComputeMatchPoints(pred, m)
			entry, _ := scoring.NewScoreEntry(
				shared.ScoreEntryID(uuid.Must(uuid.NewV7()).String()),
				pred.UserID(), pred.PoolID(),
				scoring.SourceMatch, string(m.ID()), pts,
			)
			if entry != nil {
				uc.scores.Upsert(ctx, entry)
				sr.TotalPoints += pts
			}
		}
		sr.PoolsScored++
	}

	return sr
}

func (uc *SyncResults) resolveTeam(ctx context.Context, code string, tournamentID shared.TournamentID) *team.Team {
	// Try football-data.org external ID first
	t, err := uc.teams.FindByExternalID(ctx, "footballdata", code)
	if err == nil {
		return t
	}
	// Fall back to direct code lookup
	t, err = uc.teams.FindByCode(ctx, code, tournamentID)
	if err == nil {
		return t
	}
	return nil
}
