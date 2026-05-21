package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// MatchesHandler handles match-related HTTP requests.
type MatchesHandler struct {
	matches match.Repository
}

func NewMatchesHandler(matches match.Repository) *MatchesHandler {
	return &MatchesHandler{matches: matches}
}

// ListMatches returns all matches for the current tournament.
// GET /api/v1/matches
func (h *MatchesHandler) ListMatches(c *gin.Context) {
	tournamentID := c.Query("tournament_id")
	if tournamentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tournament_id is required"})
		return
	}

	matches, err := h.matches.FindByTournament(c.Request.Context(), shared.TournamentID(tournamentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list matches"})
		return
	}

	if matches == nil {
		matches = []*match.Match{}
	}

	type matchDTO struct {
		ID           string `json:"id"`
		Stage        string `json:"stage"`
		GroupID      string `json:"group_id,omitempty"`
		HomeTeamID   string `json:"home_team_id"`
		AwayTeamID   string `json:"away_team_id"`
		KickoffAt    string `json:"kickoff_at"`
		Venue        string `json:"venue"`
		Status       string `json:"status"`
	}

	result := make([]matchDTO, 0, len(matches))
	for _, m := range matches {
		dto := matchDTO{
			ID:         string(m.ID()),
			Stage:      string(m.Stage()),
			HomeTeamID: string(m.HomeTeamID()),
			AwayTeamID: string(m.AwayTeamID()),
			KickoffAt:  m.KickoffAt().Format("2006-01-02T15:04:05Z"),
			Venue:      m.Venue(),
			Status:     string(m.Status()),
		}
		if gid := m.GroupID(); gid != nil {
			dto.GroupID = string(*gid)
		}
		result = append(result, dto)
	}

	c.JSON(http.StatusOK, gin.H{"matches": result})
}

// GetMatch returns a single match by ID.
// GET /api/v1/matches/:id
func (h *MatchesHandler) GetMatch(c *gin.Context) {
	id := c.Param("id")
	m, err := h.matches.FindByID(c.Request.Context(), shared.MatchID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "match not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          string(m.ID()),
		"stage":       string(m.Stage()),
		"home_team_id": string(m.HomeTeamID()),
		"away_team_id": string(m.AwayTeamID()),
		"kickoff_at":  m.KickoffAt().Format("2006-01-02T15:04:05Z"),
		"venue":       m.Venue(),
		"status":      string(m.Status()),
	})
}
