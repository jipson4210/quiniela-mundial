package openfootball

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
)

// Adapter implements tournament.SeedProvider using openfootball data.
type Adapter struct {
	client *Client
}

func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

func (a *Adapter) FetchSeed(ctx context.Context) (*tournament.SeedData, error) {
	raw, err := a.client.FetchWorldCupJSON(ctx)
	if err != nil {
		return nil, fmt.Errorf("openfootball adapter: %w", err)
	}

	dto, err := Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("openfootball adapter: %w", err)
	}

	return toSeedData(dto)
}

// teamNameToCode maps openfootball team names to FIFA 3-letter codes.
// Placeholder names like "1A", "W73" are padded to 3 chars.
var teamNameToCode = map[string]string{
	"Argentina":            "ARG",
	"Australia":            "AUS",
	"Austria":              "AUT",
	"Belgium":              "BEL",
	"Bosnia & Herzegovina": "BIH",
	"Brazil":               "BRA",
	"Canada":               "CAN",
	"Cape Verde":           "CPV",
	"Colombia":             "COL",
	"Croatia":              "CRO",
	"Curaçao":             "CUW",
	"Czech Republic":       "CZE",
	"DR Congo":             "COD",
	"Ecuador":              "ECU",
	"Egypt":                "EGY",
	"England":              "ENG",
	"France":               "FRA",
	"Germany":              "GER",
	"Ghana":                "GHA",
	"Haiti":                "HAI",
	"Iran":                 "IRN",
	"Iraq":                 "IRQ",
	"Ivory Coast":          "CIV",
	"Japan":                "JPN",
	"Jordan":               "JOR",
	"Mexico":               "MEX",
	"Morocco":              "MAR",
	"Netherlands":          "NED",
	"New Zealand":          "NZL",
	"Norway":               "NOR",
	"Panama":               "PAN",
	"Paraguay":             "PAR",
	"Portugal":             "POR",
	"Qatar":                "QAT",
	"Saudi Arabia":         "KSA",
	"Scotland":             "SCO",
	"Senegal":              "SEN",
	"South Africa":         "RSA",
	"South Korea":          "KOR",
	"Spain":                "ESP",
	"Sweden":               "SWE",
	"Switzerland":          "SUI",
	"Tunisia":              "TUN",
	"Turkey":               "TUR",
	"USA":                  "USA",
	"Uruguay":              "URU",
	"Uzbekistan":           "UZB",
}

func resolveCode(name string) string {
	if code, ok := teamNameToCode[name]; ok {
		return code
	}
	// Placeholder like "1A" (group winner A) → "W1A"
	// Placeholder like "2B" (group runner-up B) → "R2B"
	if len(name) == 2 {
		prefix := "W"
		if name[0] == '2' {
			prefix = "R"
		}
		return prefix + string(name[1]) + string(name[0])
	}
	// Multi-group path like "1A/B/C/D/F" → "P1A"
	if strings.Contains(name, "/") {
		return "KP" + strings.Split(name, "/")[0]
	}
	// Long placeholder like "W100" → use as-is if <= 4 chars
	raw := strings.ToUpper(strings.ReplaceAll(name, " ", ""))
	if len(raw) > 4 {
		raw = raw[:4]
	}
	return raw
}

func toSeedData(dto *worldcupDTO) (*tournament.SeedData, error) {
	if len(dto.Matches) == 0 {
		return nil, fmt.Errorf("no matches found in worldcup.json")
	}

	groupTeams := make(map[string]map[string]struct{})
	var allMatches []tournament.SeedMatch

	for _, m := range dto.Matches {
		groupName := groupFromString(m.Group)
		homeCode := resolveCode(m.Team1)
		awayCode := resolveCode(m.Team2)

		// Only track team→group membership for group-stage matches
		if groupName != "" {
			if _, ok := groupTeams[groupName]; !ok {
				groupTeams[groupName] = make(map[string]struct{})
			}
			groupTeams[groupName][homeCode] = struct{}{}
			groupTeams[groupName][awayCode] = struct{}{}
		}

		kickoff, err := parseKickoff(m.Date, m.Time)
		if err != nil {
			return nil, fmt.Errorf("invalid kickoff for match %d: %w", m.Num, err)
		}

		stage := stageFromRound(m.Round, m.Group)

		allMatches = append(allMatches, tournament.SeedMatch{
			Stage:     stage,
			GroupName: groupName,
			HomeCode:  homeCode,
			AwayCode:  awayCode,
			KickoffAt: kickoff.Format(time.RFC3339),
			Venue:     m.Ground,
		})
	}

	// Build SeedGroups
	var groups []tournament.SeedGroup
	for name, codes := range groupTeams {
		teams := make([]string, 0, len(codes))
		for code := range codes {
			teams = append(teams, code)
		}
		groups = append(groups, tournament.SeedGroup{
			Name:  name,
			Teams: teams,
		})
	}

	firstMatch := allMatches[0]
	lastMatch := allMatches[len(allMatches)-1]

	return &tournament.SeedData{
		TournamentName: dto.Name,
		StartsAt:       firstMatch.KickoffAt,
		EndsAt:         lastMatch.KickoffAt,
		Groups:         groups,
		Matches:        allMatches,
	}, nil
}

func groupFromString(raw string) string {
	name := strings.TrimPrefix(raw, "Group ")
	return strings.TrimSpace(name)
}

func stageFromRound(roundName, group string) string {
	rn := strings.ToLower(roundName)
	switch {
	case strings.Contains(rn, "matchday"):
		return "group"
	case strings.Contains(rn, "round of 32"):
		return "round_of_32"
	case strings.Contains(rn, "round of 16"):
		return "round_of_16"
	case strings.Contains(rn, "quarter"):
		return "quarter_final"
	case strings.Contains(rn, "semi"):
		return "semi_final"
	case strings.Contains(rn, "third place"):
		return "third_place"
	case strings.Contains(rn, "final"):
		return "final"
	default:
		if group != "" {
			return "group"
		}
		return "round_of_32"
	}
}

func parseKickoff(dateStr, timeStr string) (time.Time, error) {
	// openfootball format: date="2026-06-11", time="13:00 UTC-6"
	// Remove timezone suffix and parse manually
	timeStr = strings.TrimSpace(timeStr)

	// Extract timezone offset: "13:00 UTC-6" → "13:00", offset=-6
	var offsetHours int
	parts := strings.Fields(timeStr)
	if len(parts) >= 2 {
		tzPart := parts[1] // "UTC-6" or "UTC+5"
		if strings.HasPrefix(tzPart, "UTC") {
			offStr := strings.TrimPrefix(tzPart, "UTC")
			fmt.Sscanf(offStr, "%d", &offsetHours)
		}
		timeStr = parts[0]
	}

	loc := time.FixedZone("UTC"+strings.TrimPrefix(strings.Fields(timeStr+" "+strings.Join(parts[1:], " "))[0], "UTC"), offsetHours*3600)

	// Parse: "2026-06-11T13:00:00"
	combined := dateStr + "T" + timeStr + ":00"
	// Try parsing without timezone first, then apply location
	t, err := time.Parse("2006-01-02T15:04:05", combined)
	if err != nil {
		return time.Time{}, err
	}

	// Convert to UTC
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc).UTC(), nil
}
