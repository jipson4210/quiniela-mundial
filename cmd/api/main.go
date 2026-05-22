package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	// Domain ports
	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/prediction"
	"github.com/josemontalban/quiniela-mundial/internal/domain/scoring"
	"github.com/josemontalban/quiniela-mundial/internal/domain/team"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
	"github.com/josemontalban/quiniela-mundial/internal/domain/user"

	// Application
	"github.com/josemontalban/quiniela-mundial/internal/application/commands"
	"github.com/josemontalban/quiniela-mundial/internal/application/queries"

	// Infrastructure
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/auth/jwt"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/email"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/external/footballdata"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/external/openfootball"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/persistence/postgres"

	// Interfaces
	"github.com/josemontalban/quiniela-mundial/internal/interfaces/cli"
	httppkg "github.com/josemontalban/quiniela-mundial/internal/interfaces/http"
	"github.com/josemontalban/quiniela-mundial/internal/interfaces/http/handlers"
)

func main() {
	// Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Config
	viper.SetDefault("API_PORT", "8080")
	viper.SetDefault("GIN_MODE", "debug")
	viper.SetDefault("LOG_LEVEL", "debug")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_NAME", "quiniela")
	viper.SetDefault("DB_USER", "quiniela")
	viper.SetDefault("DB_PASSWORD", "quiniela_dev")
	viper.SetDefault("OPENFOOTBALL_URL", "")
	viper.SetDefault("JWT_SECRET", "change-me-in-production")
	viper.SetDefault("JWT_EXPIRATION_HOURS", "72")
	viper.SetDefault("APP_URL", "http://localhost:8080")
	viper.AutomaticEnv()

	// Subcommand routing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "seed":
			runSeed()
			return
		case "sync":
			runSync()
			return
		}
	}
	runServe()
}

// runServe starts the HTTP API server.
func runServe() {
	gin.SetMode(viper.GetString("GIN_MODE"))

	// Database connection
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		viper.GetString("DB_USER"), viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"), viper.GetString("DB_PORT"),
		viper.GetString("DB_NAME"))

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("db connect failed")
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("db ping failed")
	}
	log.Info().Msg("connected to database")

	// Repositories
	tournamentRepo := postgres.NewTournamentRepo(db)
	teamRepo := postgres.NewTeamRepo(db)
	matchRepo := postgres.NewMatchRepo(db)
	userRepo := postgres.NewUserRepo(db)
	poolRepo := postgres.NewPoolRepo(db)
	predictionRepo := postgres.NewPredictionRepo(db)
	scoreRepo := postgres.NewScoreRepo(db)
	bracketRepo := postgres.NewBracketRepo(db)

	// JWT service
	jwtExpHours := viper.GetInt("JWT_EXPIRATION_HOURS")
	if jwtExpHours <= 0 {
		jwtExpHours = 72
	}
	jwtService := jwt.NewService(
		viper.GetString("JWT_SECRET"),
		time.Duration(jwtExpHours)*time.Hour,
	)

	// Email sender (noop for dev)
	emailSender := email.NewNoopSender()

	// Application — Commands
	registerUser := commands.NewRegisterUser(userRepo)
	loginUser := commands.NewLoginUser(userRepo)
	createPool := commands.NewCreatePool(poolRepo)
	inviteMember := commands.NewInviteMember(poolRepo, emailSender, viper.GetString("APP_URL"))
	acceptInvitation := commands.NewAcceptInvitation(poolRepo)
	submitPrediction := commands.NewSubmitPrediction(predictionRepo, matchRepo, poolRepo)
	finalizeMatch := commands.NewFinalizeMatch(matchRepo)
	computeMatchPoints := commands.NewComputeMatchPoints(predictionRepo, matchRepo, scoreRepo)
	submitBracket := commands.NewSubmitBracket(bracketRepo, poolRepo, tournamentRepo)

	// Queries
	getUserPools := queries.NewGetUserPools(poolRepo)
	getRanking := queries.NewGetRanking(scoreRepo, userRepo)

	// Ensure seed command references compile
	_ = tournamentRepo
	_ = teamRepo
	_ = matchRepo
	_ = userRepo
	_ = poolRepo
	_ = tournament.Repository(nil)
	_ = match.Repository(nil)
	_ = team.Repository(nil)
	_ = user.Repository(nil)
	_ = pool.Repository(nil)
	_ = prediction.Repository(nil)
	_ = scoring.Repository(nil)

	// HTTP handlers
	authH := handlers.NewAuthHandler(registerUser, loginUser, jwtService)
	poolsH := handlers.NewPoolsHandler(createPool, inviteMember, acceptInvitation, getUserPools, getRanking)
	predictionsH := handlers.NewPredictionsHandler(submitPrediction)
	// Sync command for HTTP endpoint / admin
	fdClient := footballdata.NewClient(viper.GetString("FOOTBALL_DATA_API_KEY"))
	fdAdapter := footballdata.NewAdapter(fdClient)
	syncCmd := commands.NewSyncResults(fdAdapter, matchRepo, teamRepo, predictionRepo, scoreRepo, tournamentRepo)

	adminH := handlers.NewAdminHandler(finalizeMatch, computeMatchPoints, syncCmd)
	bracketsH := handlers.NewBracketsHandler(submitBracket)

	// Router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(corsMiddleware())

	httppkg.RegisterRoutes(router, matchRepo, authH, poolsH, predictionsH, bracketsH, adminH, jwtService)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", viper.GetString("API_PORT")),
		Handler: router,
	}

	go func() {
		log.Info().Str("port", viper.GetString("API_PORT")).Msg("API starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("forced shutdown")
	}
	log.Info().Msg("server stopped")
}

// runSeed executes the seed command.
func runSeed() {
	log.Info().Msg("seeding tournament data...")

	// DB connection
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		viper.GetString("DB_USER"), viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"), viper.GetString("DB_PORT"),
		viper.GetString("DB_NAME"))

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("db connect failed")
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("db ping failed")
	}

	openfootballClient := openfootball.NewClient(viper.GetString("OPENFOOTBALL_URL"))
	seedProvider := openfootball.NewAdapter(openfootballClient)

	tournamentsRepo := postgres.NewTournamentRepo(db)
	teamsRepo := postgres.NewTeamRepo(db)
	matchesRepo := postgres.NewMatchRepo(db)

	seedCmd := commands.NewSeedTournament(seedProvider, tournamentsRepo, teamsRepo, matchesRepo)
	cli.RunSeed(seedCmd)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// runSync executes the sync command to fetch results from external APIs.
func runSync() {
	log.Info().Msg("syncing results from external APIs...")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		viper.GetString("DB_USER"), viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"), viper.GetString("DB_PORT"),
		viper.GetString("DB_NAME"))

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("db connect failed")
	}
	defer db.Close()

	// Football-data.org adapter (primary)
	fdClient := footballdata.NewClient(viper.GetString("FOOTBALL_DATA_API_KEY"))
	fdAdapter := footballdata.NewAdapter(fdClient)

	// Repos
	tournamentRepo := postgres.NewTournamentRepo(db)
	teamRepo := postgres.NewTeamRepo(db)
	matchRepo := postgres.NewMatchRepo(db)
	predictionRepo := postgres.NewPredictionRepo(db)
	scoreRepo := postgres.NewScoreRepo(db)

	syncCmd := commands.NewSyncResults(fdAdapter, matchRepo, teamRepo, predictionRepo, scoreRepo, tournamentRepo)

	// Default: sync from yesterday to next week
	from := time.Now().AddDate(0, 0, -1)
	to := time.Now().AddDate(0, 0, 7)

	if len(os.Args) > 2 {
		if parsed, err := time.Parse("2006-01-02", os.Args[2]); err == nil {
			from = parsed
		}
	}
	if len(os.Args) > 3 {
		if parsed, err := time.Parse("2006-01-02", os.Args[3]); err == nil {
			to = parsed
		}
	}

	results, err := syncCmd.Execute(context.Background(), from, to)
	if err != nil {
		log.Fatal().Err(err).Msg("sync failed")
	}

	ok, skipped, errors := 0, 0, 0
	for _, r := range results {
		if r.Error != "" {
			errors++
		} else if r.Skipped {
			skipped++
		} else {
			ok++
		}
	}
	log.Info().Msgf("sync done: %d synced, %d skipped, %d errors", ok, skipped, errors)
}

// ensure time import is used in this file
var _ = time.Now
