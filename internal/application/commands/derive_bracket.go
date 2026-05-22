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
	// Predicted bracket: derived from user's match predictions + BracketPrediction.
	Groups       map[string]GroupTableDTO   `json:"groups"`
	RoundOf32    [16]BracketSlotDTO          `json:"round_of_32"`
	RoundOf16    [8]BracketSlotDTO           `json:"round_of_16"`
	QuarterFinal [4]BracketSlotDTO           `json:"quarter_final"`
	SemiFinal    [2]BracketSlotDTO           `json:"semi_final"`
	ThirdPlace   BracketSlotDTO              `json:"third_place"`
	Final        BracketSlotDTO              `json:"final"`

	// Actual bracket: derived from official match results once each match is
	// finished. Empty/partial until matches start finalizing.
	GroupsActual       map[string]GroupTableDTO `json:"groups_actual"`
	ActualRoundOf32    [16]BracketSlotDTO       `json:"actual_round_of_32"`
	ActualRoundOf16    [8]BracketSlotDTO        `json:"actual_round_of_16"`
	ActualQuarterFinal [4]BracketSlotDTO        `json:"actual_quarter_final"`
	ActualSemiFinal    [2]BracketSlotDTO        `json:"actual_semi_final"`
	ActualThirdPlace   BracketSlotDTO           `json:"actual_third_place"`
	ActualFinal        BracketSlotDTO           `json:"actual_final"`
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

	// Actual bracket from official match results.
	tablesActual, groupsActualDTO := uc.buildActualGroupTables(ctx, tid, groups, teamCodes)
	bracketActual := uc.buildActualKnockoutBracket(ctx, tid, tablesActual)

	log.Printf("[derive] bracket derived from %d predictions across %d groups (bracket=%v, actual_groups=%d)", len(predsByMatch), len(tables), bp != nil, len(groupsActualDTO))

	out := bracketToDTO(bracket)
	out.Groups = output.Groups
	out.GroupsActual = groupsActualDTO
	fillActualSlots(out, bracketActual)
	return out, nil
}

// buildActualGroupTables constructs the group standings from official results
// of finished matches (m.Result()). Returns the tables for downstream R32
// derivation plus a DTO map for the response.
func (uc *DeriveBracket) buildActualGroupTables(
	ctx context.Context,
	tid shared.TournamentID,
	groups []match.GroupInfo,
	teamCodes map[shared.TeamID]string,
) (map[string]tournament.GroupTable, map[string]GroupTableDTO) {
	tables := make(map[string]tournament.GroupTable, len(groups))
	dtos := make(map[string]GroupTableDTO, len(groups))

	for _, g := range groups {
		matchResults := make(map[shared.MatchID][2]int)
		matchInfos := make([]tournament.MatchInfo, 0, len(g.MatchIDs))

		for _, mid := range g.MatchIDs {
			m, err := uc.matches.FindByID(ctx, mid)
			if err != nil {
				continue
			}
			matchInfos = append(matchInfos, tournament.MatchInfo{
				MatchID:    mid,
				HomeTeamID: m.HomeTeamID(),
				AwayTeamID: m.AwayTeamID(),
			})
			if m.Status() == match.StatusFinished && m.Result() != nil {
				r := m.Result()
				matchResults[mid] = [2]int{r.HomeGoals(), r.AwayGoals()}
			}
		}

		table := tournament.BuildGroupTable(g.Teams, teamCodes, matchResults, matchInfos)
		tables[g.Name] = table
		dtos[g.Name] = GroupTableDTO{Name: g.Name, Standings: table}
	}
	return tables, dtos
}

// buildActualKnockoutBracket builds the bracket from real KO match results.
// FillRoundOf32 places teams using the FIFA pairing pattern from the actual
// standings; each round's winners come from m.Result() of the finished match.
func (uc *DeriveBracket) buildActualKnockoutBracket(
	ctx context.Context,
	tid shared.TournamentID,
	tablesActual map[string]tournament.GroupTable,
) *tournament.KnockoutBracket {
	b := &tournament.KnockoutBracket{}
	b.FillRoundOf32(tablesActual)

	apply := func(slots []tournament.KnockoutSlot, stage tournament.Stage) {
		matches, err := uc.matches.FindByStage(ctx, tid, stage)
		if err != nil {
			return
		}
		for i := range slots {
			if i >= len(matches) {
				break
			}
			m := matches[i]
			if m.Status() != match.StatusFinished {
				continue
			}
			// Trust the actual match's teams once the match has finished —
			// the slot's teams from FillRoundOf32 are predictive only.
			slots[i].HomeTeamID = m.HomeTeamID()
			slots[i].AwayTeamID = m.AwayTeamID()
			slots[i].WinnerID = winnerOfMatch(m)
		}
	}

	apply(b.RoundOf32[:], tournament.StageRoundOf32)
	apply(b.RoundOf16[:], tournament.StageRoundOf16)
	apply(b.QuarterFinals[:], tournament.StageQuarterFinal)
	apply(b.SemiFinals[:], tournament.StageSemiFinal)

	if mm, err := uc.matches.FindByStage(ctx, tid, tournament.StageThirdPlace); err == nil && len(mm) > 0 {
		m := mm[0]
		if m.Status() == match.StatusFinished {
			b.ThirdPlace = tournament.KnockoutSlot{
				HomeTeamID: m.HomeTeamID(), AwayTeamID: m.AwayTeamID(),
				HomeLabel: "Perdedor 101", AwayLabel: "Perdedor 102",
				WinnerID: winnerOfMatch(m),
			}
		}
	}
	if mm, err := uc.matches.FindByStage(ctx, tid, tournament.StageFinal); err == nil && len(mm) > 0 {
		m := mm[0]
		if m.Status() == match.StatusFinished {
			b.Final = tournament.KnockoutSlot{
				HomeTeamID: m.HomeTeamID(), AwayTeamID: m.AwayTeamID(),
				HomeLabel: "Ganador 101", AwayLabel: "Ganador 102",
				WinnerID: winnerOfMatch(m),
			}
		}
	}
	return b
}

// winnerOfMatch returns the team that won the match, considering penalties
// first, then extra time, then regular time. Returns "" if the match has no
// result yet or ended in a draw (only possible in group stage).
func winnerOfMatch(m *match.Match) shared.TeamID {
	if m == nil || m.Result() == nil {
		return ""
	}
	r := m.Result()
	if r.HomeGoalsPen() != nil && r.AwayGoalsPen() != nil {
		if *r.HomeGoalsPen() > *r.AwayGoalsPen() {
			return m.HomeTeamID()
		}
		if *r.AwayGoalsPen() > *r.HomeGoalsPen() {
			return m.AwayTeamID()
		}
	}
	if r.HomeGoalsET() != nil && r.AwayGoalsET() != nil {
		if *r.HomeGoalsET() > *r.AwayGoalsET() {
			return m.HomeTeamID()
		}
		if *r.AwayGoalsET() > *r.HomeGoalsET() {
			return m.AwayTeamID()
		}
	}
	if r.HomeGoals() > r.AwayGoals() {
		return m.HomeTeamID()
	}
	if r.AwayGoals() > r.HomeGoals() {
		return m.AwayTeamID()
	}
	return ""
}

// fillActualSlots copies the actual KnockoutBracket into the DTO's Actual* fields.
func fillActualSlots(out *DeriveBracketOutput, b *tournament.KnockoutBracket) {
	for i, s := range b.RoundOf32 {
		out.ActualRoundOf32[i] = BracketSlotDTO{string(s.HomeTeamID), s.HomeLabel, string(s.AwayTeamID), s.AwayLabel, string(s.WinnerID)}
	}
	for i, s := range b.RoundOf16 {
		out.ActualRoundOf16[i] = BracketSlotDTO{string(s.HomeTeamID), s.HomeLabel, string(s.AwayTeamID), s.AwayLabel, string(s.WinnerID)}
	}
	for i, s := range b.QuarterFinals {
		out.ActualQuarterFinal[i] = BracketSlotDTO{string(s.HomeTeamID), s.HomeLabel, string(s.AwayTeamID), s.AwayLabel, string(s.WinnerID)}
	}
	for i, s := range b.SemiFinals {
		out.ActualSemiFinal[i] = BracketSlotDTO{string(s.HomeTeamID), s.HomeLabel, string(s.AwayTeamID), s.AwayLabel, string(s.WinnerID)}
	}
	out.ActualThirdPlace = BracketSlotDTO{string(b.ThirdPlace.HomeTeamID), b.ThirdPlace.HomeLabel, string(b.ThirdPlace.AwayTeamID), b.ThirdPlace.AwayLabel, string(b.ThirdPlace.WinnerID)}
	out.ActualFinal = BracketSlotDTO{string(b.Final.HomeTeamID), b.Final.HomeLabel, string(b.Final.AwayTeamID), b.Final.AwayLabel, string(b.Final.WinnerID)}
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
