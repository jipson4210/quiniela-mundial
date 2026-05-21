package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/user"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/persistence/postgres/sqlc"
)

type UserRepo struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{q: sqlc.New(db), db: db}
}

func (r *UserRepo) Save(ctx context.Context, u *user.User) error {
	_, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           string(u.ID()),
		Email:        u.Email(),
		PasswordHash: u.PasswordHash(),
		DisplayName:  u.DisplayName(),
	})
	return err
}

func (r *UserRepo) FindByID(ctx context.Context, id shared.UserID) (*user.User, error) {
	row, err := r.q.GetUserByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toUserDomain(row), nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toUserDomain(row), nil
}

func toUserDomain(row sqlc.User) *user.User {
	return user.Reconstruct(
		shared.UserID(row.ID),
		row.Email,
		row.PasswordHash,
		row.DisplayName,
		row.VerifiedAt,
		row.CreatedAt,
	)
}
