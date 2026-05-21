package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// User represents a row from the users table.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	VerifiedAt   *time.Time
	CreatedAt    time.Time
}

type CreateUserParams struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	const sql = `INSERT INTO users (id, email, password_hash, display_name) VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, display_name, verified_at, created_at`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.Email, arg.PasswordHash, arg.DisplayName)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.VerifiedAt, &u.CreatedAt)
	return u, err
}

func (q *Queries) GetUserByID(ctx context.Context, id string) (User, error) {
	const sql = `SELECT id, email, password_hash, display_name, verified_at, created_at FROM users WHERE id = $1`
	row := q.db.QueryRow(ctx, sql, id)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.VerifiedAt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return User{}, err
	}
	return u, err
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	const sql = `SELECT id, email, password_hash, display_name, verified_at, created_at FROM users WHERE email = $1`
	row := q.db.QueryRow(ctx, sql, email)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.VerifiedAt, &u.CreatedAt)
	return u, err
}

func (q *Queries) ListUsers(ctx context.Context) ([]User, error) {
	const sql = `SELECT id, email, password_hash, display_name, verified_at, created_at FROM users ORDER BY created_at DESC`
	rows, err := q.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.VerifiedAt, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
