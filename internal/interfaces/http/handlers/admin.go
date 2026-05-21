package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
)

// AdminHandler handles admin operations (finalize matches, trigger scoring).
type AdminHandler struct {
	finalizeMatch    *commands.FinalizeMatch
	computeMatchPoints *commands.ComputeMatchPoints
}

func NewAdminHandler(finalizeMatch *commands.FinalizeMatch, computeMatchPoints *commands.ComputeMatchPoints) *AdminHandler {
	return &AdminHandler{finalizeMatch: finalizeMatch, computeMatchPoints: computeMatchPoints}
}

// FinalizeMatch sets the official result for a match and triggers scoring.
// POST /api/v1/admin/matches/:id/finalize
func (h *AdminHandler) FinalizeMatch(c *gin.Context) {
	matchID := c.Param("id")

	var req struct {
		HomeGoals          int    `json:"home_goals" binding:"min=0"`
		AwayGoals          int    `json:"away_goals" binding:"min=0"`
		HomeGoalsET        *int   `json:"home_goals_et"`
		AwayGoalsET        *int   `json:"away_goals_et"`
		HomeGoalsPenalties *int   `json:"home_goals_penalties"`
		AwayGoalsPenalties *int   `json:"away_goals_penalties"`
		PoolID             string `json:"pool_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Step 1: Finalize the match
	_, err := h.finalizeMatch.Execute(c.Request.Context(), commands.FinalizeMatchInput{
		MatchID:            matchID,
		HomeGoals:          req.HomeGoals,
		AwayGoals:          req.AwayGoals,
		HomeGoalsET:        req.HomeGoalsET,
		AwayGoalsET:        req.AwayGoalsET,
		HomeGoalsPenalties: req.HomeGoalsPenalties,
		AwayGoalsPenalties: req.AwayGoalsPenalties,
		Source:             "manual",
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Step 2: Compute points for all predictions in this pool
	result, err := h.computeMatchPoints.Execute(c.Request.Context(), commands.ComputeMatchPointsInput{
		MatchID: matchID,
		PoolID:  req.PoolID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"finalized": result})
}
