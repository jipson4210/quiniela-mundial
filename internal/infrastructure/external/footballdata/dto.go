package footballdata

import "time"

// matchesResponse is the top-level response from /v4/matches.
type matchesResponse struct {
	Matches []matchDTO `json:"matches"`
}

// matchDTO mirrors a match from football-data.org v4 API.
type matchDTO struct {
	ID        int        `json:"id"`
	Status    string     `json:"status"`    // "FINISHED", "IN_PLAY", "SCHEDULED", etc.
	Stage     string     `json:"stage"`     // "GROUP_STAGE", "ROUND_OF_32", etc.
	Group     string     `json:"group"`     // "GROUP_A"
	UTCDate   time.Time  `json:"utcDate"`
	HomeTeam  teamDTO    `json:"homeTeam"`
	AwayTeam  teamDTO    `json:"awayTeam"`
	Score     scoreDTO   `json:"score"`
}

type teamDTO struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	TLA       string `json:"tla"` // 3-letter acronym
	Crest     string `json:"crest"`
}

type scoreDTO struct {
	Winner    string        `json:"winner"`    // "HOME_TEAM", "AWAY_TEAM", "DRAW", null
	Duration  string        `json:"duration"`  // "REGULAR", "EXTRA_TIME", "PENALTY_SHOOTOUT"
	FullTime  scoreValuesDTO `json:"fullTime"`
	HalfTime  scoreValuesDTO `json:"halfTime"`
	ExtraTime *scoreValuesDTO `json:"extraTime"`
	Penalties *scoreValuesDTO `json:"penalties"`
}

type scoreValuesDTO struct {
	Home *int `json:"home"`
	Away *int `json:"away"`
}
