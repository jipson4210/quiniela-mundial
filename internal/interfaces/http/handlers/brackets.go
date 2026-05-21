package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
)

// BracketsHandler handles bracket prediction HTTP requests.
type BracketsHandler struct {
	submitBracket *commands.SubmitBracket
}

func NewBracketsHandler(submitBracket *commands.SubmitBracket) *BracketsHandler {
	return &BracketsHandler{submitBracket: submitBracket}
}

// SubmitBracket creates or updates a bracket prediction for a pool.
// POST /api/v1/pools/:id/bracket
func (h *BracketsHandler) SubmitBracket(c *gin.Context) {
	poolID := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		TournamentID        string   `json:"tournament_id" binding:"required"`
		TeamsToRoundOf32    []string `json:"teams_to_round_of_32" binding:"required,len=32"`
		TeamsToRoundOf16    []string `json:"teams_to_round_of_16" binding:"required,len=16"`
		TeamsToQuarterFinal []string `json:"teams_to_quarter_final" binding:"required,len=8"`
		TeamsToSemiFinal    []string `json:"teams_to_semi_final" binding:"required,len=4"`
		TeamsToFinal        []string `json:"teams_to_final" binding:"required,len=2"`
		ThirdPlaceWinner    string   `json:"third_place_winner" binding:"required"`
		Champion            string   `json:"champion" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.submitBracket.Execute(c.Request.Context(), commands.SubmitBracketInput{
		UserID:              userID,
		PoolID:              poolID,
		TournamentID:        req.TournamentID,
		TeamsToRoundOf32:    req.TeamsToRoundOf32,
		TeamsToRoundOf16:    req.TeamsToRoundOf16,
		TeamsToQuarterFinal: req.TeamsToQuarterFinal,
		TeamsToSemiFinal:    req.TeamsToSemiFinal,
		TeamsToFinal:        req.TeamsToFinal,
		ThirdPlaceWinner:    req.ThirdPlaceWinner,
		Champion:            req.Champion,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"bracket": output})
}
