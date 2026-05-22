package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/team"
)

// TeamsHandler handles team-related HTTP requests.
type TeamsHandler struct {
	teams team.Repository
}

func NewTeamsHandler(teams team.Repository) *TeamsHandler {
	return &TeamsHandler{teams: teams}
}

// ListTeams returns all teams for a tournament.
// GET /api/v1/teams?tournament_id=
func (h *TeamsHandler) ListTeams(c *gin.Context) {
	tournamentID := c.Query("tournament_id")
	if tournamentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tournament_id is required"})
		return
	}

	teams, err := h.teams.FindByTournament(c.Request.Context(), shared.TournamentID(tournamentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list teams"})
		return
	}

	type teamDTO struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}

	result := make([]teamDTO, 0, len(teams))
	for _, t := range teams {
		result = append(result, teamDTO{
			ID:   string(t.ID()),
			Code: t.Code(),
			Name: t.Name(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"teams": result})
}
