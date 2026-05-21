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

	// Infrastructure
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/auth/jwt"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/email"
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
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		runSeed()
		return
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
	poolsH := handlers.NewPoolsHandler(createPool, inviteMember, acceptInvitation)
	predictionsH := handlers.NewPredictionsHandler(submitPrediction)
	adminH := handlers.NewAdminHandler(finalizeMatch, computeMatchPoints)
	bracketsH := handlers.NewBracketsHandler(submitBracket)

	// Router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

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
