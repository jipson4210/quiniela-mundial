package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
)

// PredictionsHandler handles match prediction HTTP requests.
type PredictionsHandler struct {
	submitPrediction *commands.SubmitPrediction
}

func NewPredictionsHandler(submitPrediction *commands.SubmitPrediction) *PredictionsHandler {
	return &PredictionsHandler{submitPrediction: submitPrediction}
}

// SubmitPrediction creates or updates a match prediction for a pool.
// POST /api/v1/pools/:id/predictions
func (h *PredictionsHandler) SubmitPrediction(c *gin.Context) {
	poolID := c.Param("id")

	var req struct {
		MatchID   string `json:"match_id" binding:"required"`
		HomeGoals int    `json:"home_goals" binding:"min=0,max=30"`
		AwayGoals int    `json:"away_goals" binding:"min=0,max=30"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	output, err := h.submitPrediction.Execute(c.Request.Context(), commands.SubmitPredictionInput{
		UserID:    userID,
		PoolID:    poolID,
		MatchID:   req.MatchID,
		HomeGoals: req.HomeGoals,
		AwayGoals: req.AwayGoals,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"prediction": output})
}
