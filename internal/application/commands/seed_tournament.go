package commands

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/team"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
)

// SeedTournament orchestrates the tournament bootstrap: download worldcup.json,
// parse into domain objects, and persist everything.
type SeedTournament struct {
	seedProvider tournament.SeedProvider
	tournaments  tournament.Repository
	teams        team.Repository
	matches      match.Repository
}

func NewSeedTournament(
	seedProvider tournament.SeedProvider,
	tournaments tournament.Repository,
	teams team.Repository,
	matches match.Repository,
) *SeedTournament {
	return &SeedTournament{
		seedProvider: seedProvider,
		tournaments:  tournaments,
		teams:        teams,
		matches:      matches,
	}
}

// Execute runs the seed operation. It is idempotent: if the tournament already
// exists, it logs and skips.
func (uc *SeedTournament) Execute(ctx context.Context) error {
	log.Println("[seed] fetching tournament data from provider...")
	data, err := uc.seedProvider.FetchSeed(ctx)
	if err != nil {
		return fmt.Errorf("seed: fetch: %w", err)
	}

	// Check if tournament already seeded
	existing, err := uc.tournaments.FindCurrent(ctx)
	if err == nil && existing != nil {
		log.Printf("[seed] tournament %q already exists (id=%s), skipping", existing.Name(), existing.ID())
		return nil
	}

	// Parse dates
	startsAt, err := time.Parse(time.RFC3339, data.StartsAt)
	if err != nil {
		return fmt.Errorf("seed: parse starts_at: %w", err)
	}
	endsAt, err := time.Parse(time.RFC3339, data.EndsAt)
	if err != nil {
		return fmt.Errorf("seed: parse ends_at: %w", err)
	}

	// Create and persist tournament first (teams FK reference it)
	tournamentID := shared.TournamentID(uuid.Must(uuid.NewV7()).String())
	t, err := tournament.New(tournamentID, data.TournamentName, startsAt, endsAt)
	if err != nil {
		return fmt.Errorf("seed: create tournament: %w", err)
	}
	if err := uc.tournaments.Save(ctx, t); err != nil {
		return fmt.Errorf("seed: save tournament: %w", err)
	}

	// Seed teams
	teamIDsByCode := make(map[string]shared.TeamID, 64)
	groupIDsByName := make(map[string]shared.GroupID, 12)

	log.Printf("[seed] creating %d groups and teams...", len(data.Groups))

	for _, g := range data.Groups {
		groupID := shared.GroupID(uuid.Must(uuid.NewV7()).String())
		groupIDsByName[g.Name] = groupID

		teamIDs := make([]shared.TeamID, 0, 4)
		for _, code := range g.Teams {
			teamID := shared.TeamID(uuid.Must(uuid.NewV7()).String())
			teamIDsByCode[code] = teamID
			teamIDs = append(teamIDs, teamID)

			tm, err := team.New(teamID, code, code, "", "", tournamentID)
			if err != nil {
				return fmt.Errorf("seed: create team %s: %w", code, err)
			}
			if err := uc.teams.Save(ctx, tm); err != nil {
				return fmt.Errorf("seed: save team %s: %w", code, err)
			}

			// Store external ID mapping for openfootball
			if err := uc.teams.SaveExternalID(ctx, team.NewExternalID(teamID, "openfootball", code)); err != nil {
				return fmt.Errorf("seed: save external id for %s: %w", code, err)
			}
		}

		domainGroup := tournament.NewGroup(groupID, g.Name, teamIDs)
		if err := t.AddGroup(domainGroup); err != nil {
			return fmt.Errorf("seed: add group %s: %w", g.Name, err)
		}

		// Persist stage_group and team memberships
		if err := uc.matches.CreateStageGroup(ctx, tournamentID, groupID, g.Name); err != nil {
			return fmt.Errorf("seed: save stage_group %s: %w", g.Name, err)
		}
		for _, teamID := range teamIDs {
			if err := uc.matches.AddTeamToGroup(ctx, groupID, teamID); err != nil {
				return fmt.Errorf("seed: add team to group: %w", err)
			}
		}
	}

	// Create placeholder teams for knockout matches (e.g., "R2A", "W1B", "KP1A")
	for _, sm := range data.Matches {
		for _, code := range []string{sm.HomeCode, sm.AwayCode} {
			if _, exists := teamIDsByCode[code]; !exists {
				teamID := shared.TeamID(uuid.Must(uuid.NewV7()).String())
				teamIDsByCode[code] = teamID
				tm, err := team.New(teamID, code, code, "", "", tournamentID)
				if err != nil {
					return fmt.Errorf("seed: create knockout team %s: %w", code, err)
				}
				if err := uc.teams.Save(ctx, tm); err != nil {
					return fmt.Errorf("seed: save knockout team %s: %w", code, err)
				}
			}
		}
	}

	log.Printf("[seed] created %d teams total", len(teamIDsByCode))

	// Seed stage definitions with scoring rules
	for _, st := range tournament.AllStages {
		if st == tournament.StageGroup {
			continue // no bracket points for group stage
		}
		points := tournament.DefaultStagePoints[st]
		t.AddStage(tournament.NewStageDef(st, points, string(st)))
	}

	// Persist tournament
	if err := uc.tournaments.Save(ctx, t); err != nil {
		return fmt.Errorf("seed: save tournament: %w", err)
	}

	// Seed matches
	log.Printf("[seed] creating %d matches...", len(data.Matches))
	for i, sm := range data.Matches {
		matchID := shared.MatchID(uuid.Must(uuid.NewV7()).String())

		homeID, ok := teamIDsByCode[sm.HomeCode]
		if !ok {
			return fmt.Errorf("seed: match %d: unknown home team code %q", i, sm.HomeCode)
		}
		awayID, ok := teamIDsByCode[sm.AwayCode]
		if !ok {
			return fmt.Errorf("seed: match %d: unknown away team code %q", i, sm.AwayCode)
		}

		kickoffAt, err := time.Parse(time.RFC3339, sm.KickoffAt)
		if err != nil {
			return fmt.Errorf("seed: match %d: parse kickoff: %w", i, err)
		}

		var groupID *shared.GroupID
		if sm.GroupName != "" {
			gid := groupIDsByName[sm.GroupName]
			groupID = &gid
		}

		m, err := match.New(matchID, tournamentID, tournament.Stage(sm.Stage), groupID, homeID, awayID, kickoffAt, sm.Venue)
		if err != nil {
			return fmt.Errorf("seed: match %d: %w", i, err)
		}
		if err := uc.matches.Save(ctx, m); err != nil {
			return fmt.Errorf("seed: save match %d: %w", i, err)
		}
	}

	log.Printf("[seed] done! tournament=%s, teams=%d, matches=%d", t.Name(), len(teamIDsByCode), len(data.Matches))
	return nil
}
