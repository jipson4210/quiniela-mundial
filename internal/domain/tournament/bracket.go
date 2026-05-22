package tournament

import (
	"sort"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// GroupStanding represents a team's position in a group.
type GroupStanding struct {
	TeamID       shared.TeamID `json:"team_id"`
	TeamCode     string        `json:"team_code"`
	GroupName    string        `json:"group_name"`
	Played       int           `json:"played"`
	Won          int           `json:"won"`
	Drawn        int           `json:"drawn"`
	Lost         int           `json:"lost"`
	GoalsFor     int           `json:"goals_for"`
	GoalsAgainst int           `json:"goals_against"`
	GoalDiff     int           `json:"goal_diff"`
	Points       int           `json:"points"`
}

type GroupTable []GroupStanding

func (gt GroupTable) Len() int      { return len(gt) }
func (gt GroupTable) Swap(i, j int) { gt[i], gt[j] = gt[j], gt[i] }
func (gt GroupTable) Less(i, j int) bool {
	if gt[i].Points != gt[j].Points {
		return gt[i].Points > gt[j].Points
	}
	if gt[i].GoalDiff != gt[j].GoalDiff {
		return gt[i].GoalDiff > gt[j].GoalDiff
	}
	if gt[i].GoalsFor != gt[j].GoalsFor {
		return gt[i].GoalsFor > gt[j].GoalsFor
	}
	return gt[i].TeamCode < gt[j].TeamCode
}

type MatchInfo struct {
	MatchID    shared.MatchID
	HomeTeamID shared.TeamID
	AwayTeamID shared.TeamID
}

// BuildGroupTable computes group standings from match predictions.
func BuildGroupTable(
	teams []shared.TeamID,
	teamCodes map[shared.TeamID]string,
	matchResults map[shared.MatchID][2]int,
	matches []MatchInfo,
) GroupTable {
	standings := make(map[shared.TeamID]*GroupStanding)
	for _, tid := range teams {
		code := teamCodes[tid]
		if code == "" {
			code = string(tid)[:4]
		}
		standings[tid] = &GroupStanding{TeamID: tid, TeamCode: code}
	}

	for _, m := range matches {
		home := standings[m.HomeTeamID]
		away := standings[m.AwayTeamID]
		if home == nil || away == nil {
			continue
		}

		result, ok := matchResults[m.MatchID]
		if !ok {
			continue
		}
		homeGoals := result[0]
		awayGoals := result[1]

		home.Played++
		away.Played++
		home.GoalsFor += homeGoals
		home.GoalsAgainst += awayGoals
		away.GoalsFor += awayGoals
		away.GoalsAgainst += homeGoals
		home.GoalDiff = home.GoalsFor - home.GoalsAgainst
		away.GoalDiff = away.GoalsFor - away.GoalsAgainst

		if homeGoals > awayGoals {
			home.Won++
			home.Points += 3
			away.Lost++
		} else if homeGoals < awayGoals {
			away.Won++
			away.Points += 3
			home.Lost++
		} else {
			home.Drawn++
			home.Points++
			away.Drawn++
			away.Points++
		}
	}

	table := make(GroupTable, 0, len(standings))
	for _, s := range standings {
		table = append(table, *s)
	}
	sort.Sort(table)
	return table
}

// KnockoutBracket represents the full tournament bracket from R32 to Champion.
type KnockoutBracket struct {
	// Round of 32: 16 matches (73-88), pre-filled from group positions
	RoundOf32 [16]KnockoutSlot
	// Round of 16: 8 matches (89-96), filled from R32 winners
	RoundOf16 [8]KnockoutSlot
	// Quarter finals: 4 matches (97-100)
	QuarterFinals [4]KnockoutSlot
	// Semi finals: 2 matches (101-102)
	SemiFinals [2]KnockoutSlot
	// Third place: 1 match (103) — losers of semifinals
	ThirdPlace KnockoutSlot
	// Final: 1 match (104) — winners of semifinals
	Final KnockoutSlot
}

// KnockoutSlot holds a matchup in the bracket.
type KnockoutSlot struct {
	HomeTeamID shared.TeamID
	AwayTeamID shared.TeamID
	HomeLabel  string // "Primero A", "Ganador 73", etc.
	AwayLabel  string // "Segundo B", "Ganador 75", etc.
	WinnerID   shared.TeamID // set after prediction
}

// FillRoundOf32 populates the 16 R32 slots from group standings.
// Follows FIFA 2026 bracket pattern.
func (b *KnockoutBracket) FillRoundOf32(tables map[string]GroupTable) {
	type slot struct {
		idx      int
		homePos  string // "1A", "2B", etc.
		awayPos  string
	}
	// FIFA 2026 Round of 32 pairings (simplified from official pattern)
	pairings := [16]slot{
		{0, "2A", "2B"},       // 73
		{1, "1E", "3ABCD_F"},  // 74
		{2, "1F", "2C"},       // 75
		{3, "1C", "2F"},       // 76
		{4, "1I", "3CDFGH"},   // 77
		{5, "2E", "2I"},       // 78
		{6, "1A", "3CEFHI"},   // 79
		{7, "1L", "3EHIJK"},   // 80
		{8, "1D", "3BEFIJ"},   // 81
		{9, "1G", "3AEHIJ"},   // 82
		{10, "2K", "2L"},       // 83
		{11, "1H", "2J"},       // 84
		{12, "1B", "3EFGIJ"},  // 85
		{13, "1J", "2H"},       // 86
		{14, "1K", "3DEIJL"},  // 87
		{15, "2D", "2G"},       // 88
	}

	for _, p := range pairings {
		homeID, homeLabel := resolveSlot(tables, p.homePos)
		awayID, awayLabel := resolveSlot(tables, p.awayPos)
		b.RoundOf32[p.idx] = KnockoutSlot{
			HomeTeamID: homeID,
			AwayTeamID: awayID,
			HomeLabel:  homeLabel,
			AwayLabel:  awayLabel,
		}
	}
}

// resolveSlot finds the team for a given bracket slot label.
// Labels: "1A" = winner of group A, "2B" = runner-up of group B, "3ABCD_F" = best 3rd from A/B/C/D/F
func resolveSlot(tables map[string]GroupTable, label string) (shared.TeamID, string) {
	if len(label) == 2 {
		pos := label[0] // '1' or '2'
		grp := string(label[1])
		table := tables[grp]
		if pos == '1' && len(table) >= 1 {
			return table[0].TeamID, table[0].TeamCode
		}
		if pos == '2' && len(table) >= 2 {
			return table[1].TeamID, table[1].TeamCode
		}
	}
	// For 3rd place slots, find the best 3rd from specified groups
	if len(label) > 3 && label[0] == '3' {
		return bestThirdPlace(tables, label), label
	}
	return "", label
}

// bestThirdPlace finds the best 3rd place team from specified groups.
func bestThirdPlace(tables map[string]GroupTable, label string) shared.TeamID {
	// Parse group letters from label like "3ABCD_F" or "3CEFHI"
	var candidates []GroupStanding
	for _, g := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"} {
		if containsGroup(label, g) {
			table := tables[g]
			if len(table) >= 3 {
				candidates = append(candidates, table[2])
			}
		}
	}
	// Sort by points, GD, GF
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].Points < candidates[j].Points ||
				(candidates[i].Points == candidates[j].Points && candidates[i].GoalDiff < candidates[j].GoalDiff) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	if len(candidates) > 0 {
		return candidates[0].TeamID
	}
	return ""
}

func containsGroup(label, group string) bool {
	for _, c := range label {
		if string(c) == group {
			return true
		}
	}
	return false
}

// FillNextRound advances winners from one round to the next.
func FillNextRound(from []KnockoutSlot, to []KnockoutSlot, startMatchID int) {
	for i := range to {
		if i*2 < len(from) && i*2+1 < len(from) {
			m1 := startMatchID + i*2
			m2 := startMatchID + i*2 + 1
			to[i] = KnockoutSlot{
				HomeTeamID: from[i*2].WinnerID,
				AwayTeamID: from[i*2+1].WinnerID,
				HomeLabel:  winnerLabel(m1),
				AwayLabel:  winnerLabel(m2),
			}
		}
	}
}

func winnerLabel(matchID int) string {
	return "Ganador " + intToStr(matchID)
}

func loserLabel(matchID int) string {
	return "Perdedor " + intToStr(matchID)
}

func intToStr(n int) string {
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// FillThirdPlace sets the third place match from semifinal losers.
func (b *KnockoutBracket) FillThirdPlace(sfLoser1, sfLoser2 shared.TeamID) {
	b.ThirdPlace = KnockoutSlot{
		HomeTeamID: sfLoser1,
		AwayTeamID: sfLoser2,
		HomeLabel:  loserLabel(101),
		AwayLabel:  loserLabel(102),
	}
}

// FillFinal sets the final from semifinal winners.
func (b *KnockoutBracket) FillFinal() {
	b.Final = KnockoutSlot{
		HomeTeamID: b.SemiFinals[0].WinnerID,
		AwayTeamID: b.SemiFinals[1].WinnerID,
		HomeLabel:  winnerLabel(101),
		AwayLabel:  winnerLabel(102),
	}
}

// Champion returns the winner of the final.
func (b *KnockoutBracket) Champion() shared.TeamID {
	return b.Final.WinnerID
}

// ThirdPlaceWinner returns the winner of the third place match.
func (b *KnockoutBracket) ThirdPlaceWinner() shared.TeamID {
	return b.ThirdPlace.WinnerID
}
