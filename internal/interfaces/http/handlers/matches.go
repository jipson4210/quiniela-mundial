package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

type MatchesHandler struct {
	matches match.Repository
}

func NewMatchesHandler(matches match.Repository) *MatchesHandler {
	return &MatchesHandler{matches: matches}
}

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
		ID         string `json:"id"`
		Stage      string `json:"stage"`
		GroupID    string `json:"group_id,omitempty"`
		HomeTeamID string `json:"home_team_id"`
		AwayTeamID string `json:"away_team_id"`
		KickoffAt  string `json:"kickoff_at"`
		Venue      string `json:"venue"`
		Status     string `json:"status"`
		HomeGoals  *int   `json:"home_goals,omitempty"`
		AwayGoals  *int   `json:"away_goals,omitempty"`
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
		if m.Result() != nil {
			h := m.Result().HomeGoals()
			a := m.Result().AwayGoals()
			dto.HomeGoals = &h
			dto.AwayGoals = &a
		}
		result = append(result, dto)
	}

	c.JSON(http.StatusOK, gin.H{"matches": result})
}

func (h *MatchesHandler) GetMatch(c *gin.Context) {
	id := c.Param("id")
	m, err := h.matches.FindByID(c.Request.Context(), shared.MatchID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "match not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           string(m.ID()),
		"stage":        string(m.Stage()),
		"home_team_id": string(m.HomeTeamID()),
		"away_team_id": string(m.AwayTeamID()),
		"kickoff_at":   m.KickoffAt().Format("2006-01-02T15:04:05Z"),
		"venue":        m.Venue(),
		"status":       string(m.Status()),
	})
}
