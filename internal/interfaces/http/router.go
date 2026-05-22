package http

import (
	"github.com/gin-gonic/gin"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/auth/jwt"
	"github.com/josemontalban/quiniela-mundial/internal/interfaces/http/handlers"
	"github.com/josemontalban/quiniela-mundial/internal/interfaces/http/middleware"
)

// RegisterRoutes wires all HTTP routes to their handlers.
func RegisterRoutes(
	router *gin.Engine,
	matchesRepo match.Repository,
	teamsH *handlers.TeamsHandler,
	authH *handlers.AuthHandler,
	poolsH *handlers.PoolsHandler,
	predictionsH *handlers.PredictionsHandler,
	bracketsH *handlers.BracketsHandler,
	adminH *handlers.AdminHandler,
	jwtService *jwt.Service,
) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")
	{
		// Public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/login", authH.Login)
		}

		// Public match routes
		matchesH := handlers.NewMatchesHandler(matchesRepo)
		v1.GET("/matches", matchesH.ListMatches)
		v1.GET("/matches/:id", matchesH.GetMatch)
		v1.GET("/teams", teamsH.ListTeams)

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(jwtService))
		{
			protected.GET("/pools", poolsH.ListPools)
			protected.POST("/pools", poolsH.CreatePool)
			protected.POST("/pools/:id/invitations", poolsH.InviteMember)
			protected.GET("/pools/:id/ranking", poolsH.GetRanking)
			protected.POST("/pools/:id/predictions", predictionsH.SubmitPrediction)
			protected.GET("/pools/:id/predictions", predictionsH.ListMyPredictions)
			protected.GET("/pools/:id/bracket/derived", bracketsH.DeriveBracket)
			protected.POST("/pools/:id/bracket", bracketsH.SubmitBracket)
			protected.POST("/invitations/:token/accept", poolsH.AcceptInvitation)
			protected.POST("/admin/matches/:id/finalize", adminH.FinalizeMatch)
			protected.POST("/admin/sync", adminH.SyncResults)
			protected.POST("/admin/bracket/stage", adminH.ComputeBracketStage)
		}
	}
}
