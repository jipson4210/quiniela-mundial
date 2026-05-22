package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
)

// AdminHandler handles admin operations (finalize matches, trigger scoring, sync, bracket).
type AdminHandler struct {
	finalizeMatch       *commands.FinalizeMatch
	computeMatchPoints  *commands.ComputeMatchPoints
	syncResults         *commands.SyncResults
	computeBracketPoints *commands.ComputeBracketPoints
}

func NewAdminHandler(
	finalizeMatch *commands.FinalizeMatch,
	computeMatchPoints *commands.ComputeMatchPoints,
	syncResults *commands.SyncResults,
	computeBracketPoints *commands.ComputeBracketPoints,
) *AdminHandler {
	return &AdminHandler{
		finalizeMatch:       finalizeMatch,
		computeMatchPoints:  computeMatchPoints,
		syncResults:         syncResults,
		computeBracketPoints: computeBracketPoints,
	}
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

// SyncResults triggers external result sync for a date range.
// POST /api/v1/admin/sync
func (h *AdminHandler) SyncResults(c *gin.Context) {
	var req struct {
		From string `json:"from"` // "2026-06-11"
		To   string `json:"to"`   // "2026-06-11"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	from, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		from = time.Now().AddDate(0, 0, -1)
	}
	to, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		to = time.Now().AddDate(0, 0, 7)
	}

	results, err := h.syncResults.Execute(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"synced": len(results)})
}

// ComputeBracketStage awards bracket points for a tournament phase.
// POST /api/v1/admin/bracket/stage
func (h *AdminHandler) ComputeBracketStage(c *gin.Context) {
	var req struct {
		PoolID      string   `json:"pool_id" binding:"required"`
		Stage       string   `json:"stage" binding:"required"`     // round_of_32, round_of_16, quarter_final, semi_final, final, third_place, champion
		ActualTeams []string `json:"actual_teams"`                 // team IDs that reached this stage
		Champion    string   `json:"champion"`                     // for champion stage
		ThirdPlace  string   `json:"third_place"`                  // for third_place stage
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.computeBracketPoints.Execute(c.Request.Context(), commands.BracketStageInput{
		PoolID:      req.PoolID,
		Stage:       tournament.Stage(req.Stage),
		ActualTeams: req.ActualTeams,
		Champion:    req.Champion,
		ThirdPlace:  req.ThirdPlace,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bracket_scoring": output})
}
