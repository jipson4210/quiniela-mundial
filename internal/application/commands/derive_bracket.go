package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/team"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
)

type DeriveBracketInput struct {
	PoolID       string
	TournamentID string
}

type BracketSlotDTO struct {
	HomeTeamID string `json:"home_team_id"`
	HomeLabel  string `json:"home_label"`
	AwayTeamID string `json:"away_team_id"`
	AwayLabel  string `json:"away_label"`
	WinnerID   string `json:"winner_id,omitempty"`
}

type DeriveBracketOutput struct {
	Groups       map[string]GroupTableDTO   `json:"groups"`
	RoundOf32    [16]BracketSlotDTO          `json:"round_of_32"`
	RoundOf16    [8]BracketSlotDTO           `json:"round_of_16"`
	QuarterFinal [4]BracketSlotDTO           `json:"quarter_final"`
	SemiFinal    [2]BracketSlotDTO           `json:"semi_final"`
	ThirdPlace   BracketSlotDTO              `json:"third_place"`
	Final        BracketSlotDTO              `json:"final"`
}

type GroupTableDTO struct {
	Name      string                       `json:"name"`
	Standings []tournament.GroupStanding   `json:"standings"`
}

type DeriveBracket struct {
	predictions prediction.Repository
	matches     match.Repository
	teams       team.Repository
}

func NewDeriveBracket(pred prediction.Repository, m match.Repository, t team.Repository) *DeriveBracket {
	return &DeriveBracket{predictions: pred, matches: m, teams: t}
}

func (uc *DeriveBracket) Execute(ctx context.Context, input DeriveBracketInput) (*DeriveBracketOutput, error) {
	poolID := shared.PoolID(input.PoolID)
	tid := shared.TournamentID(input.TournamentID)

	groups, err := uc.matches.GetGroupStructure(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("derive: groups: %w", err)
	}

	allTeams, err := uc.teams.FindByTournament(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("derive: teams: %w", err)
	}
	teamCodes := make(map[shared.TeamID]string)
	for _, t := range allTeams {
		teamCodes[t.ID()] = t.Code()
	}

	predsByMatch := make(map[shared.MatchID][2]int)
	for _, g := range groups {
		for _, mid := range g.MatchIDs {
			preds, _ := uc.predictions.FindByPoolAndMatch(ctx, poolID, mid)
			if len(preds) > 0 {
				predsByMatch[mid] = [2]int{preds[0].HomeGoals(), preds[0].AwayGoals()}
			}
		}
	}

	tables := make(map[string]tournament.GroupTable)
	output := &DeriveBracketOutput{Groups: make(map[string]GroupTableDTO)}

	for _, g := range groups {
		matchResults := make(map[shared.MatchID][2]int)
		matchInfos := make([]tournament.MatchInfo, 0, len(g.MatchIDs))

		for _, mid := range g.MatchIDs {
			if result, ok := predsByMatch[mid]; ok {
				matchResults[mid] = result
			}
			m, err := uc.matches.FindByID(ctx, mid)
			if err == nil {
				matchInfos = append(matchInfos, tournament.MatchInfo{
					MatchID:    mid,
					HomeTeamID: m.HomeTeamID(),
					AwayTeamID: m.AwayTeamID(),
				})
			}
		}

		table := tournament.BuildGroupTable(g.Teams, teamCodes, matchResults, matchInfos)
		tables[g.Name] = table
		output.Groups[g.Name] = GroupTableDTO{Name: g.Name, Standings: table}
	}

	bracket := &tournament.KnockoutBracket{}
	bracket.FillRoundOf32(tables)

	// Propagate R32 winners from predictions
	r32Matches, _ := uc.matches.FindByStage(ctx, tid, tournament.StageRoundOf32)
	for i := range bracket.RoundOf32 {
		if i >= len(r32Matches) {
			break
		}
		preds, _ := uc.predictions.FindByPoolAndMatch(ctx, poolID, r32Matches[i].ID())
		if len(preds) > 0 && preds[0].HomeGoals() != preds[0].AwayGoals() {
			m, _ := uc.matches.FindByID(ctx, r32Matches[i].ID())
			if m != nil {
				if preds[0].HomeGoals() > preds[0].AwayGoals() {
					bracket.RoundOf32[i].WinnerID = m.HomeTeamID()
				} else {
					bracket.RoundOf32[i].WinnerID = m.AwayTeamID()
				}
			}
		}
	}

	tournament.FillNextRound(bracket.RoundOf32[:], bracket.RoundOf16[:], 89)
	tournament.FillNextRound(bracket.RoundOf16[:], bracket.QuarterFinals[:], 97)
	tournament.FillNextRound(bracket.QuarterFinals[:], bracket.SemiFinals[:], 101)

	log.Printf("[derive] bracket derived from %d predictions across %d groups", len(predsByMatch), len(tables))

	return bracketToDTO(bracket), nil
}

func bracketToDTO(b *tournament.KnockoutBracket) *DeriveBracketOutput {
	dto := &DeriveBracketOutput{Groups: make(map[string]GroupTableDTO)}
	for i, s := range b.RoundOf32 {
		dto.RoundOf32[i] = BracketSlotDTO{string(s.HomeTeamID), s.HomeLabel, string(s.AwayTeamID), s.AwayLabel, string(s.WinnerID)}
	}
	for i, s := range b.RoundOf16 {
		dto.RoundOf16[i] = BracketSlotDTO{string(s.HomeTeamID), s.HomeLabel, string(s.AwayTeamID), s.AwayLabel, string(s.WinnerID)}
	}
	for i, s := range b.QuarterFinals {
		dto.QuarterFinal[i] = BracketSlotDTO{string(s.HomeTeamID), s.HomeLabel, string(s.AwayTeamID), s.AwayLabel, string(s.WinnerID)}
	}
	for i, s := range b.SemiFinals {
		dto.SemiFinal[i] = BracketSlotDTO{string(s.HomeTeamID), s.HomeLabel, string(s.AwayTeamID), s.AwayLabel, string(s.WinnerID)}
	}
	dto.ThirdPlace = BracketSlotDTO{string(b.ThirdPlace.HomeTeamID), b.ThirdPlace.HomeLabel, string(b.ThirdPlace.AwayTeamID), b.ThirdPlace.AwayLabel, string(b.ThirdPlace.WinnerID)}
	dto.Final = BracketSlotDTO{string(b.Final.HomeTeamID), b.Final.HomeLabel, string(b.Final.AwayTeamID), b.Final.AwayLabel, string(b.Final.WinnerID)}
	return dto
}
