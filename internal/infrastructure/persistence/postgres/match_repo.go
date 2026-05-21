package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/josemontalban/quiniela-mundial/internal/domain/match"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/tournament"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/persistence/postgres/sqlc"
)

type MatchRepo struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

func NewMatchRepo(db *pgxpool.Pool) *MatchRepo {
	return &MatchRepo{q: sqlc.New(db), db: db}
}

func (r *MatchRepo) Save(ctx context.Context, m *match.Match) error {
	var groupID *string
	if gid := m.GroupID(); gid != nil {
		s := string(*gid)
		groupID = &s
	}

	_, err := r.q.CreateMatch(ctx, sqlc.CreateMatchParams{
		ID:           string(m.ID()),
		TournamentID: string(m.TournamentID()),
		Stage:        string(m.Stage()),
		GroupID:      groupID,
		HomeTeamID:   string(m.HomeTeamID()),
		AwayTeamID:   string(m.AwayTeamID()),
		KickoffAt:    m.KickoffAt(),
		Venue:        m.Venue(),
		Status:       string(m.Status()),
	})
	return err
}

func (r *MatchRepo) SaveBatch(ctx context.Context, matches []*match.Match) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)
	for _, m := range matches {
		var groupID *string
		if gid := m.GroupID(); gid != nil {
			s := string(*gid)
			groupID = &s
		}
		_, err := q.CreateMatch(ctx, sqlc.CreateMatchParams{
			ID:           string(m.ID()),
			TournamentID: string(m.TournamentID()),
			Stage:        string(m.Stage()),
			GroupID:      groupID,
			HomeTeamID:   string(m.HomeTeamID()),
			AwayTeamID:   string(m.AwayTeamID()),
			KickoffAt:    m.KickoffAt(),
			Venue:        m.Venue(),
			Status:       string(m.Status()),
		})
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *MatchRepo) FindByID(ctx context.Context, id shared.MatchID) (*match.Match, error) {
	row, err := r.q.GetMatchByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toMatchDomain(row), nil
}

func (r *MatchRepo) FindByTournament(ctx context.Context, tournamentID shared.TournamentID) ([]*match.Match, error) {
	rows, err := r.q.ListMatchesByTournament(ctx, string(tournamentID))
	if err != nil {
		return nil, err
	}
	matches := make([]*match.Match, 0, len(rows))
	for _, row := range rows {
		matches = append(matches, toMatchDomain(row))
	}
	return matches, nil
}

func (r *MatchRepo) FindByStage(ctx context.Context, tournamentID shared.TournamentID, stage tournament.Stage) ([]*match.Match, error) {
	rows, err := r.q.ListMatchesByStage(ctx, sqlc.ListMatchesByStageParams{
		TournamentID: string(tournamentID),
		Stage:        string(stage),
	})
	if err != nil {
		return nil, err
	}
	matches := make([]*match.Match, 0, len(rows))
	for _, row := range rows {
		matches = append(matches, toMatchDomain(row))
	}
	return matches, nil
}

func (r *MatchRepo) FindByTeamAndKickoff(ctx context.Context, homeTeamID, awayTeamID shared.TeamID, kickoffAt time.Time) (*match.Match, error) {
	// Simplified: FindByID is used instead in the seed use case
	return nil, shared.ErrNotFound
}

func (r *MatchRepo) CreateStageGroup(ctx context.Context, tournamentID shared.TournamentID, groupID shared.GroupID, name string) error {
	_, err := r.q.CreateStageGroup(ctx, sqlc.CreateStageGroupParams{
		ID:           string(groupID),
		TournamentID: string(tournamentID),
		Name:         name,
	})
	return err
}

func (r *MatchRepo) AddTeamToGroup(ctx context.Context, groupID shared.GroupID, teamID shared.TeamID) error {
	return r.q.AddTeamToGroup(ctx, sqlc.AddTeamToGroupParams{
		GroupID: string(groupID),
		TeamID:  string(teamID),
	})
}

func (r *MatchRepo) UpdateResult(ctx context.Context, m *match.Match) error {
	res := m.Result()
	if res == nil {
		return nil
	}
	return r.q.UpdateMatchResult(ctx, sqlc.UpdateMatchResultParams{
		ID:           string(m.ID()),
		HomeGoals:    res.HomeGoals(),
		AwayGoals:    res.AwayGoals(),
		HomeGoalsEt:  res.HomeGoalsET(),
		AwayGoalsEt:  res.AwayGoalsET(),
		HomeGoalsPen: res.HomeGoalsPen(),
		AwayGoalsPen: res.AwayGoalsPen(),
		ResultSource: string(res.Source()),
		FinalizedAt:  res.FinalizedAt(),
	})
}

func toMatchDomain(row sqlc.Match) *match.Match {
	var groupID *shared.GroupID
	if row.GroupID != nil {
		gid := shared.GroupID(*row.GroupID)
		groupID = &gid
	}

	var result *match.Result
	if row.Status == "finished" && row.HomeGoals != nil {
		result = reconstructResult(row)
	}

	return match.Reconstruct(
		shared.MatchID(row.ID),
		shared.TournamentID(row.TournamentID),
		tournament.Stage(row.Stage),
		groupID,
		shared.TeamID(row.HomeTeamID),
		shared.TeamID(row.AwayTeamID),
		row.KickoffAt,
		row.Venue,
		match.Status(row.Status),
		result,
		row.CreatedAt,
	)
}

func reconstructResult(row sqlc.Match) *match.Result {
	var src match.ResultSource
	if row.ResultSource != nil {
		src = match.ResultSource(*row.ResultSource)
	}
	var finalizedAt time.Time
	if row.FinalizedAt != nil {
		finalizedAt = *row.FinalizedAt
	}
	homeGoals := 0
	if row.HomeGoals != nil {
		homeGoals = *row.HomeGoals
	}
	awayGoals := 0
	if row.AwayGoals != nil {
		awayGoals = *row.AwayGoals
	}
	return match.ReconstructResult(homeGoals, awayGoals, row.HomeGoalsEt, row.AwayGoalsEt, row.HomeGoalsPen, row.AwayGoalsPen, src, finalizedAt)
}
