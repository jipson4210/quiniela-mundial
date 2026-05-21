package tournament

import "context"

// SeedData is the domain-neutral representation of tournament bootstrap data
// returned by any seed provider (openfootball, manual JSON, etc.).
type SeedData struct {
	TournamentName string
	StartsAt       string // RFC 3339
	EndsAt         string // RFC 3339
	Groups         []SeedGroup
	Matches        []SeedMatch
}

// SeedGroup maps a World Cup group name to its 4 team codes.
type SeedGroup struct {
	Name  string   // "A".."L"
	Teams []string // FIFA codes: ["MEX","FRA","JPN","KSA"]
}

// SeedMatch represents a single match from the seed source.
type SeedMatch struct {
	Stage      string
	GroupName  string // empty for knockout
	HomeCode   string
	AwayCode   string
	KickoffAt  string // RFC 3339
	Venue      string
}

// SeedProvider is the domain port for tournament bootstrap data.
// Implemented by openfootball adapter (and potentially other sources).
type SeedProvider interface {
	FetchSeed(ctx context.Context) (*SeedData, error)
}
