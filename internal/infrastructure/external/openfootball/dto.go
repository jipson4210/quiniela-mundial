package openfootball

// worldcupDTO mirrors the actual openfootball worldcup.json structure.
// The JSON has a flat "matches" array, not nested rounds.

type worldcupDTO struct {
	Name    string     `json:"name"`
	Matches []matchDTO `json:"matches"`
}

type matchDTO struct {
	Num    int    `json:"num"`
	Round  string `json:"round"`  // "Matchday 1"
	Date   string `json:"date"`   // "2026-06-11"
	Time   string `json:"time"`   // "13:00 UTC-6"
	Team1  string `json:"team1"`  // plain string, not object
	Team2  string `json:"team2"`  // plain string, not object
	Group  string `json:"group"`  // "Group A"
	Ground string `json:"ground"` // "Mexico City"
}
