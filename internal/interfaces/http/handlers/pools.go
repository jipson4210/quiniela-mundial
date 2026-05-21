package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
)

// PoolsHandler handles pool-related HTTP requests.
type PoolsHandler struct {
	createPool      *commands.CreatePool
	inviteMember    *commands.InviteMember
	acceptInvitation *commands.AcceptInvitation
}

func NewPoolsHandler(
	createPool *commands.CreatePool,
	inviteMember *commands.InviteMember,
	acceptInvitation *commands.AcceptInvitation,
) *PoolsHandler {
	return &PoolsHandler{
		createPool:      createPool,
		inviteMember:    inviteMember,
		acceptInvitation: acceptInvitation,
	}
}

// CreatePool creates a new pool.
// POST /api/v1/pools
func (h *PoolsHandler) CreatePool(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required,min=3,max=80"`
		Description  string `json:"description"`
		TournamentID string `json:"tournament_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id") // set by JWT middleware

	output, err := h.createPool.Execute(c.Request.Context(), commands.CreatePoolInput{
		Name:         req.Name,
		Description:  req.Description,
		CreatorID:    userID,
		TournamentID: req.TournamentID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"pool": output})
}

// InviteMember creates an invitation to join a pool.
// POST /api/v1/pools/:id/invitations
func (h *PoolsHandler) InviteMember(c *gin.Context) {
	poolID := c.Param("id")

	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	output, err := h.inviteMember.Execute(c.Request.Context(), commands.InviteMemberInput{
		PoolID:    poolID,
		Email:     req.Email,
		InvitedBy: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"invitation": output})
}

// AcceptInvitation redeems an invitation token.
// POST /api/v1/invitations/:token/accept
func (h *PoolsHandler) AcceptInvitation(c *gin.Context) {
	token := c.Param("token")
	userID := c.GetString("user_id")

	output, err := h.acceptInvitation.Execute(c.Request.Context(), commands.AcceptInvitationInput{
		Token:  token,
		UserID: userID,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accepted": output})
}
