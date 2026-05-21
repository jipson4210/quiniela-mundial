package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/auth/jwt"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	register   *commands.RegisterUser
	login      *commands.LoginUser
	jwtService *jwt.Service
}

func NewAuthHandler(register *commands.RegisterUser, login *commands.LoginUser, jwtService *jwt.Service) *AuthHandler {
	return &AuthHandler{register: register, login: login, jwtService: jwtService}
}

// Register creates a new user account.
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=8"`
		DisplayName string `json:"display_name" binding:"required,min=2,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.register.Execute(c.Request.Context(), commands.RegisterUserInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Generate JWT
	token, err := h.jwtService.Generate(output.UserID, output.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":  output,
		"token": token,
	})
}

// Login authenticates a user and returns a JWT.
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.login.Execute(c.Request.Context(), commands.LoginUserInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := h.jwtService.Generate(output.UserID, output.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  output,
		"token": token,
	})
}
