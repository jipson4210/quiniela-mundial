package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Tournament struct {
	ID        string
	Name      string
	StartsAt  time.Time
	EndsAt    time.Time
	CreatedAt time.Time
}

type CreateTournamentParams struct {
	ID       string
	Name     string
	StartsAt time.Time
	EndsAt   time.Time
}

func (q *Queries) CreateTournament(ctx context.Context, arg CreateTournamentParams) (Tournament, error) {
	const sql = `INSERT INTO tournaments (id, name, starts_at, ends_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET name = $2, starts_at = $3, ends_at = $4
		RETURNING id, name, starts_at, ends_at, created_at`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.Name, arg.StartsAt, arg.EndsAt)
	var t Tournament
	err := row.Scan(&t.ID, &t.Name, &t.StartsAt, &t.EndsAt, &t.CreatedAt)
	return t, err
}

func (q *Queries) GetTournamentByID(ctx context.Context, id string) (Tournament, error) {
	const sql = `SELECT id, name, starts_at, ends_at, created_at FROM tournaments WHERE id = $1`
	row := q.db.QueryRow(ctx, sql, id)
	var t Tournament
	err := row.Scan(&t.ID, &t.Name, &t.StartsAt, &t.EndsAt, &t.CreatedAt)
	if err == pgx.ErrNoRows {
		return Tournament{}, err
	}
	return t, err
}

func (q *Queries) GetCurrentTournament(ctx context.Context) (Tournament, error) {
	const sql = `SELECT id, name, starts_at, ends_at, created_at FROM tournaments ORDER BY starts_at DESC LIMIT 1`
	row := q.db.QueryRow(ctx, sql)
	var t Tournament
	err := row.Scan(&t.ID, &t.Name, &t.StartsAt, &t.EndsAt, &t.CreatedAt)
	return t, err
}

func (q *Queries) GetTournaments(ctx context.Context) ([]Tournament, error) {
	const sql = `SELECT id, name, starts_at, ends_at, created_at FROM tournaments ORDER BY starts_at DESC`
	rows, err := q.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tournaments []Tournament
	for rows.Next() {
		var t Tournament
		if err := rows.Scan(&t.ID, &t.Name, &t.StartsAt, &t.EndsAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		tournaments = append(tournaments, t)
	}
	return tournaments, rows.Err()
}
