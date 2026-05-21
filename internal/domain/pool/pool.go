// Package pool implements the Pool aggregate root, PoolMember value object,
// Invitation entity, and PoolSettings for the Quiniela bounded context.
package pool

import (
	"context"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// Role defines the authority level of a pool member.
type Role string

const (
	RoleCreator Role = "creator"
	RoleAdmin   Role = "admin"
	RoleMember  Role = "member"
)

// Pool is the aggregate root for a private prediction group.
type Pool struct {
	id          shared.PoolID
	name        string
	description string
	creatorID   shared.UserID
	tournamentID shared.TournamentID
	createdAt   time.Time
}

// New creates a Pool with validation.
func New(id shared.PoolID, name, description string, creatorID shared.UserID, tournamentID shared.TournamentID) (*Pool, error) {
	if len(name) < 3 || len(name) > 80 {
		return nil, shared.ErrInvalidInput
	}
	return &Pool{
		id:          id,
		name:        name,
		description: description,
		creatorID:   creatorID,
		tournamentID: tournamentID,
	}, nil
}

// Reconstruct hydrates a Pool from persistence.
func Reconstruct(id shared.PoolID, name, description string, creatorID shared.UserID, tournamentID shared.TournamentID, createdAt time.Time) *Pool {
	return &Pool{
		id: id, name: name, description: description,
		creatorID: creatorID, tournamentID: tournamentID, createdAt: createdAt,
	}
}

// Accessors
func (p *Pool) ID() shared.PoolID           { return p.id }
func (p *Pool) Name() string                 { return p.name }
func (p *Pool) Description() string          { return p.description }
func (p *Pool) CreatorID() shared.UserID     { return p.creatorID }
func (p *Pool) TournamentID() shared.TournamentID { return p.tournamentID }
func (p *Pool) CreatedAt() time.Time         { return p.createdAt }

// PoolMember is a value object representing a user's membership in a pool.
type PoolMember struct {
	poolID    shared.PoolID
	userID    shared.UserID
	role      Role
	joinedAt  time.Time
	invitedBy *shared.UserID
}

func NewPoolMember(poolID shared.PoolID, userID shared.UserID, role Role, invitedBy *shared.UserID) PoolMember {
	return PoolMember{poolID: poolID, userID: userID, role: role, invitedBy: invitedBy}
}

func (pm PoolMember) PoolID() shared.PoolID   { return pm.poolID }
func (pm PoolMember) UserID() shared.UserID   { return pm.userID }
func (pm PoolMember) Role() Role              { return pm.role }
func (pm PoolMember) JoinedAt() time.Time     { return pm.joinedAt }
func (pm PoolMember) InvitedBy() *shared.UserID { return pm.invitedBy }

// Invitation represents a pending invitation to join a pool.
type Invitation struct {
	id         shared.InvitationID
	poolID     shared.PoolID
	email      string
	token      string
	invitedBy  shared.UserID
	expiresAt  time.Time
	acceptedAt *time.Time
	createdAt  time.Time
}

func NewInvitation(id shared.InvitationID, poolID shared.PoolID, email, token string, invitedBy shared.UserID, expiresAt time.Time) (*Invitation, error) {
	if email == "" {
		return nil, shared.ErrInvalidInput
	}
	if token == "" {
		return nil, shared.ErrInvalidInput
	}
	return &Invitation{
		id: id, poolID: poolID, email: email,
		token: token, invitedBy: invitedBy, expiresAt: expiresAt,
	}, nil
}

func ReconstructInvitation(id shared.InvitationID, poolID shared.PoolID, email, token string, invitedBy shared.UserID, expiresAt time.Time, acceptedAt *time.Time, createdAt time.Time) *Invitation {
	return &Invitation{
		id: id, poolID: poolID, email: email, token: token,
		invitedBy: invitedBy, expiresAt: expiresAt, acceptedAt: acceptedAt, createdAt: createdAt,
	}
}

// Accessors
func (i *Invitation) ID() shared.InvitationID    { return i.id }
func (i *Invitation) PoolID() shared.PoolID       { return i.poolID }
func (i *Invitation) Email() string               { return i.email }
func (i *Invitation) Token() string               { return i.token }
func (i *Invitation) InvitedBy() shared.UserID    { return i.invitedBy }
func (i *Invitation) ExpiresAt() time.Time        { return i.expiresAt }
func (i *Invitation) AcceptedAt() *time.Time      { return i.acceptedAt }
func (i *Invitation) CreatedAt() time.Time        { return i.createdAt }

// IsExpired returns true if the invitation has expired.
func (i *Invitation) IsExpired(at time.Time) bool {
	return at.After(i.expiresAt)
}

// IsAccepted returns true if the invitation has been accepted.
func (i *Invitation) IsAccepted() bool {
	return i.acceptedAt != nil
}

// Repository defines persistence for pools.
type Repository interface {
	Save(ctx context.Context, p *Pool) error
	FindByID(ctx context.Context, id shared.PoolID) (*Pool, error)
	FindByUser(ctx context.Context, userID shared.UserID) ([]*Pool, error)
	AddMember(ctx context.Context, pm PoolMember) error
	FindMembers(ctx context.Context, poolID shared.PoolID) ([]PoolMember, error)
	SaveInvitation(ctx context.Context, i *Invitation) error
	FindInvitationByToken(ctx context.Context, token string) (*Invitation, error)
	AcceptInvitation(ctx context.Context, invitationID shared.InvitationID, at time.Time) error
}
