package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/persistence/postgres/sqlc"
)

type PoolRepo struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

func NewPoolRepo(db *pgxpool.Pool) *PoolRepo {
	return &PoolRepo{q: sqlc.New(db), db: db}
}

func (r *PoolRepo) Save(ctx context.Context, p *pool.Pool) error {
	desc := stringOrNil(p.Description())
	_, err := r.q.CreatePool(ctx, sqlc.CreatePoolParams{
		ID:           string(p.ID()),
		Name:         p.Name(),
		Description:  desc,
		CreatorID:    string(p.CreatorID()),
		TournamentID: string(p.TournamentID()),
	})
	return err
}

func (r *PoolRepo) FindByID(ctx context.Context, id shared.PoolID) (*pool.Pool, error) {
	row, err := r.q.GetPoolByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toPoolDomain(row), nil
}

func (r *PoolRepo) FindByUser(ctx context.Context, userID shared.UserID) ([]*pool.Pool, error) {
	rows, err := r.q.ListPoolsByUser(ctx, string(userID))
	if err != nil {
		return nil, err
	}
	pools := make([]*pool.Pool, 0, len(rows))
	for _, row := range rows {
		pools = append(pools, toPoolDomain(row))
	}
	return pools, nil
}

func (r *PoolRepo) AddMember(ctx context.Context, pm pool.PoolMember) error {
	var invitedBy *string
	if ib := pm.InvitedBy(); ib != nil {
		s := string(*ib)
		invitedBy = &s
	}
	return r.q.AddPoolMember(ctx, sqlc.AddPoolMemberParams{
		PoolID:    string(pm.PoolID()),
		UserID:    string(pm.UserID()),
		Role:      string(pm.Role()),
		JoinedAt:  time.Now(),
		InvitedBy: invitedBy,
	})
}

func (r *PoolRepo) FindMembers(ctx context.Context, poolID shared.PoolID) ([]pool.PoolMember, error) {
	rows, err := r.q.GetPoolMembers(ctx, string(poolID))
	if err != nil {
		return nil, err
	}
	members := make([]pool.PoolMember, 0, len(rows))
	for _, row := range rows {
		var ib *shared.UserID
		if row.InvitedBy != nil {
			uid := shared.UserID(*row.InvitedBy)
			ib = &uid
		}
		members = append(members, pool.NewPoolMember(
			shared.PoolID(row.PoolID),
			shared.UserID(row.UserID),
			pool.Role(row.Role),
			ib,
		))
	}
	return members, nil
}

func (r *PoolRepo) SaveInvitation(ctx context.Context, i *pool.Invitation) error {
	_, err := r.q.CreateInvitation(ctx, sqlc.CreateInvitationParams{
		ID:        string(i.ID()),
		PoolID:    string(i.PoolID()),
		Email:     i.Email(),
		Token:     i.Token(),
		InvitedBy: string(i.InvitedBy()),
		ExpiresAt: i.ExpiresAt(),
	})
	return err
}

func (r *PoolRepo) FindInvitationByToken(ctx context.Context, token string) (*pool.Invitation, error) {
	row, err := r.q.GetInvitationByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return toInvitationDomain(row), nil
}

func (r *PoolRepo) AcceptInvitation(ctx context.Context, invitationID shared.InvitationID, at time.Time) error {
	return r.q.AcceptInvitation(ctx, sqlc.AcceptInvitationParams{
		ID:         string(invitationID),
		AcceptedAt: at,
	})
}

func toPoolDomain(row sqlc.Pool) *pool.Pool {
	desc := ""
	if row.Description != nil {
		desc = *row.Description
	}
	return pool.Reconstruct(
		shared.PoolID(row.ID),
		row.Name,
		desc,
		shared.UserID(row.CreatorID),
		shared.TournamentID(row.TournamentID),
		row.CreatedAt,
	)
}

func toInvitationDomain(row sqlc.Invitation) *pool.Invitation {
	return pool.ReconstructInvitation(
		shared.InvitationID(row.ID),
		shared.PoolID(row.PoolID),
		row.Email,
		row.Token,
		shared.UserID(row.InvitedBy),
		row.ExpiresAt,
		row.AcceptedAt,
		row.CreatedAt,
	)
}

func stringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
