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
	UserID       string
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
	brackets    prediction.BracketRepository
	matches     match.Repository
	teams       team.Repository
}

func NewDeriveBracket(pred prediction.Repository, brackets prediction.BracketRepository, m match.Repository, t team.Repository) *DeriveBracket {
	return &DeriveBracket{predictions: pred, brackets: brackets, matches: m, teams: t}
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

	// Load the user's BracketPrediction (winner-only picks for the knockout stage).
	// If the user hasn't submitted a bracket yet, we still return R32 derived from
	// group standings — the rest of the rounds will come empty for the UI to fill.
	var bp *prediction.BracketPrediction
	if input.UserID != "" {
		bp, _ = uc.brackets.FindByUserAndPool(ctx, shared.UserID(input.UserID), poolID)
	}

	if bp != nil {
		propagateWinners(bracket, bp)
	} else {
		// Without a bracket prediction we can still propagate empty slots so the UI
		// renders the structure of the knockout stage.
		tournament.FillNextRound(bracket.RoundOf32[:], bracket.RoundOf16[:], 89)
		tournament.FillNextRound(bracket.RoundOf16[:], bracket.QuarterFinals[:], 97)
		tournament.FillNextRound(bracket.QuarterFinals[:], bracket.SemiFinals[:], 101)
	}

	log.Printf("[derive] bracket derived from %d predictions across %d groups (bracket=%v)", len(predsByMatch), len(tables), bp != nil)

	out := bracketToDTO(bracket)
	out.Groups = output.Groups
	return out, nil
}

// propagateWinners walks the bracket round-by-round, setting WinnerID for each
// slot based on the user's BracketPrediction picks, and auto-fills the Final
// from the SF winners and the third-place match from the SF losers.
func propagateWinners(bracket *tournament.KnockoutBracket, bp *prediction.BracketPrediction) {
	r16Set := teamSet(bp.TeamsToRoundOf16())
	qfSet := teamSet(bp.TeamsToQuarterFinal())
	sfSet := teamSet(bp.TeamsToSemiFinal())
	finalSet := teamSet(bp.TeamsToFinal())

	pickWinner(bracket.RoundOf32[:], r16Set)
	tournament.FillNextRound(bracket.RoundOf32[:], bracket.RoundOf16[:], 89)

	pickWinner(bracket.RoundOf16[:], qfSet)
	tournament.FillNextRound(bracket.RoundOf16[:], bracket.QuarterFinals[:], 97)

	pickWinner(bracket.QuarterFinals[:], sfSet)
	tournament.FillNextRound(bracket.QuarterFinals[:], bracket.SemiFinals[:], 101)

	pickWinner(bracket.SemiFinals[:], finalSet)

	// Third-place match: losers of the two semifinals.
	bracket.FillThirdPlace(loserOf(bracket.SemiFinals[0]), loserOf(bracket.SemiFinals[1]))
	bracket.ThirdPlace.WinnerID = bp.ThirdPlaceWinner()

	// Final: winners of the two semifinals.
	bracket.FillFinal()
	bracket.Final.WinnerID = bp.Champion()
}

func teamSet(ids []shared.TeamID) map[shared.TeamID]struct{} {
	set := make(map[shared.TeamID]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// pickWinner sets WinnerID for each slot to whichever team (Home or Away) is in
// the advancing-teams set. Leaves WinnerID empty if neither side is present
// (e.g., the user hasn't picked, or the slot itself has empty team IDs).
func pickWinner(slots []tournament.KnockoutSlot, advancing map[shared.TeamID]struct{}) {
	for i := range slots {
		s := &slots[i]
		if s.HomeTeamID != "" {
			if _, ok := advancing[s.HomeTeamID]; ok {
				s.WinnerID = s.HomeTeamID
				continue
			}
		}
		if s.AwayTeamID != "" {
			if _, ok := advancing[s.AwayTeamID]; ok {
				s.WinnerID = s.AwayTeamID
			}
		}
	}
}

// loserOf returns the team in the slot that did not win.
func loserOf(s tournament.KnockoutSlot) shared.TeamID {
	if s.WinnerID == "" {
		return ""
	}
	if s.WinnerID == s.HomeTeamID {
		return s.AwayTeamID
	}
	return s.HomeTeamID
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
