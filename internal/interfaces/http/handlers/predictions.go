package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// PredictionsHandler handles match prediction HTTP requests.
type PredictionsHandler struct {
	submitPrediction *commands.SubmitPrediction
	predictions      prediction.Repository
}

func NewPredictionsHandler(submitPrediction *commands.SubmitPrediction, predictions prediction.Repository) *PredictionsHandler {
	return &PredictionsHandler{submitPrediction: submitPrediction, predictions: predictions}
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

// PredictionDTO is the shape returned by GET /pools/:id/predictions.
type PredictionDTO struct {
	MatchID   string `json:"match_id"`
	HomeGoals int    `json:"home_goals"`
	AwayGoals int    `json:"away_goals"`
}

// ListMyPredictions returns all match predictions submitted by the authenticated
// user in the given pool.
// GET /api/v1/pools/:id/predictions
func (h *PredictionsHandler) ListMyPredictions(c *gin.Context) {
	poolID := c.Param("id")
	userID := c.GetString("user_id")

	preds, err := h.predictions.FindByUserAndPool(c.Request.Context(), shared.UserID(userID), shared.PoolID(poolID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]PredictionDTO, 0, len(preds))
	for _, p := range preds {
		out = append(out, PredictionDTO{
			MatchID:   string(p.MatchID()),
			HomeGoals: p.HomeGoals(),
			AwayGoals: p.AwayGoals(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"predictions": out})
}
