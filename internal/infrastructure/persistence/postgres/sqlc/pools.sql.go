package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Pool struct {
	ID           string
	Name         string
	Description  *string
	CreatorID    string
	TournamentID string
	CreatedAt    time.Time
}

type PoolMember struct {
	PoolID      string
	UserID      string
	Role        string
	JoinedAt    time.Time
	InvitedBy   *string
	Email       *string
	DisplayName *string
}

type Invitation struct {
	ID         string
	PoolID     string
	Email      string
	Token      string
	InvitedBy  string
	ExpiresAt  time.Time
	AcceptedAt *time.Time
	CreatedAt  time.Time
	PoolName   *string
}

// --- pools ---

type CreatePoolParams struct {
	ID           string
	Name         string
	Description  *string
	CreatorID    string
	TournamentID string
}

func (q *Queries) CreatePool(ctx context.Context, arg CreatePoolParams) (Pool, error) {
	const sql = `INSERT INTO pools (id, name, description, creator_id, tournament_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, creator_id, tournament_id, created_at`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.Name, arg.Description, arg.CreatorID, arg.TournamentID)
	var p Pool
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.CreatorID, &p.TournamentID, &p.CreatedAt)
	return p, err
}

func (q *Queries) GetPoolByID(ctx context.Context, id string) (Pool, error) {
	const sql = `SELECT id, name, description, creator_id, tournament_id, created_at FROM pools WHERE id = $1`
	row := q.db.QueryRow(ctx, sql, id)
	var p Pool
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.CreatorID, &p.TournamentID, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return Pool{}, err
	}
	return p, err
}

func (q *Queries) ListPoolsByUser(ctx context.Context, userID string) ([]Pool, error) {
	const sql = `SELECT p.id, p.name, p.description, p.creator_id, p.tournament_id, p.created_at
		FROM pools p JOIN pool_members pm ON pm.pool_id = p.id
		WHERE pm.user_id = $1 ORDER BY p.created_at DESC`
	rows, err := q.db.Query(ctx, sql, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pools []Pool
	for rows.Next() {
		var p Pool
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatorID, &p.TournamentID, &p.CreatedAt); err != nil {
			return nil, err
		}
		pools = append(pools, p)
	}
	return pools, rows.Err()
}

// --- pool members ---

type AddPoolMemberParams struct {
	PoolID    string
	UserID    string
	Role      string
	JoinedAt  time.Time
	InvitedBy *string
}

func (q *Queries) AddPoolMember(ctx context.Context, arg AddPoolMemberParams) error {
	const sql = `INSERT INTO pool_members (pool_id, user_id, role, joined_at, invited_by)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`
	_, err := q.db.Exec(ctx, sql, arg.PoolID, arg.UserID, arg.Role, arg.JoinedAt, arg.InvitedBy)
	return err
}

func (q *Queries) GetPoolMembers(ctx context.Context, poolID string) ([]PoolMember, error) {
	const sql = `SELECT pm.pool_id, pm.user_id, pm.role, pm.joined_at, pm.invited_by, u.email, u.display_name
		FROM pool_members pm JOIN users u ON u.id = pm.user_id
		WHERE pm.pool_id = $1 ORDER BY pm.joined_at`
	rows, err := q.db.Query(ctx, sql, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []PoolMember
	for rows.Next() {
		var m PoolMember
		if err := rows.Scan(&m.PoolID, &m.UserID, &m.Role, &m.JoinedAt, &m.InvitedBy, &m.Email, &m.DisplayName); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

type GetPoolMemberParams struct {
	PoolID string
	UserID string
}

func (q *Queries) GetPoolMember(ctx context.Context, arg GetPoolMemberParams) (PoolMember, error) {
	const sql = `SELECT pool_id, user_id, role, joined_at, invited_by FROM pool_members WHERE pool_id = $1 AND user_id = $2`
	row := q.db.QueryRow(ctx, sql, arg.PoolID, arg.UserID)
	var m PoolMember
	err := row.Scan(&m.PoolID, &m.UserID, &m.Role, &m.JoinedAt, &m.InvitedBy)
	return m, err
}

// --- invitations ---

type CreateInvitationParams struct {
	ID        string
	PoolID    string
	Email     string
	Token     string
	InvitedBy string
	ExpiresAt time.Time
}

func (q *Queries) CreateInvitation(ctx context.Context, arg CreateInvitationParams) (Invitation, error) {
	const sql = `INSERT INTO invitations (id, pool_id, email, token, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, pool_id, email, token, invited_by, expires_at, accepted_at, created_at`
	row := q.db.QueryRow(ctx, sql, arg.ID, arg.PoolID, arg.Email, arg.Token, arg.InvitedBy, arg.ExpiresAt)
	var i Invitation
	err := row.Scan(&i.ID, &i.PoolID, &i.Email, &i.Token, &i.InvitedBy, &i.ExpiresAt, &i.AcceptedAt, &i.CreatedAt)
	return i, err
}

func (q *Queries) GetInvitationByToken(ctx context.Context, token string) (Invitation, error) {
	const sql = `SELECT id, pool_id, email, token, invited_by, expires_at, accepted_at, created_at FROM invitations WHERE token = $1`
	row := q.db.QueryRow(ctx, sql, token)
	var i Invitation
	err := row.Scan(&i.ID, &i.PoolID, &i.Email, &i.Token, &i.InvitedBy, &i.ExpiresAt, &i.AcceptedAt, &i.CreatedAt)
	return i, err
}

type AcceptInvitationParams struct {
	ID         string
	AcceptedAt time.Time
}

func (q *Queries) AcceptInvitation(ctx context.Context, arg AcceptInvitationParams) error {
	const sql = `UPDATE invitations SET accepted_at = $2 WHERE id = $1`
	_, err := q.db.Exec(ctx, sql, arg.ID, arg.AcceptedAt)
	return err
}

func (q *Queries) ListInvitationsByPool(ctx context.Context, poolID string) ([]Invitation, error) {
	const sql = `SELECT id, pool_id, email, token, invited_by, expires_at, accepted_at, created_at FROM invitations WHERE pool_id = $1 ORDER BY created_at DESC`
	rows, err := q.db.Query(ctx, sql, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var invs []Invitation
	for rows.Next() {
		var i Invitation
		if err := rows.Scan(&i.ID, &i.PoolID, &i.Email, &i.Token, &i.InvitedBy, &i.ExpiresAt, &i.AcceptedAt, &i.CreatedAt); err != nil {
			return nil, err
		}
		invs = append(invs, i)
	}
	return invs, rows.Err()
}

func (q *Queries) ListPendingInvitationsByEmail(ctx context.Context, email string) ([]Invitation, error) {
	const sql = `SELECT i.id, i.pool_id, i.email, i.token, i.invited_by, i.expires_at, i.accepted_at, i.created_at, p.name AS pool_name
		FROM invitations i JOIN pools p ON p.id = i.pool_id
		WHERE i.email = $1 AND i.accepted_at IS NULL AND i.expires_at > now()`
	rows, err := q.db.Query(ctx, sql, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var invs []Invitation
	for rows.Next() {
		var i Invitation
		if err := rows.Scan(&i.ID, &i.PoolID, &i.Email, &i.Token, &i.InvitedBy, &i.ExpiresAt, &i.AcceptedAt, &i.CreatedAt, &i.PoolName); err != nil {
			return nil, err
		}
		invs = append(invs, i)
	}
	return invs, rows.Err()
}
