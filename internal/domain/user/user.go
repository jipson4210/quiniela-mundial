package user

import (
	"context"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// User is the aggregate root for identity and access.
type User struct {
	id           shared.UserID
	email        string
	passwordHash string
	displayName  string
	verifiedAt   *time.Time
	createdAt    time.Time
}

// New creates a User with validation.
func New(id shared.UserID, email, passwordHash, displayName string) (*User, error) {
	if email == "" || len(email) > 255 {
		return nil, shared.ErrInvalidInput
	}
	if len(displayName) < 2 || len(displayName) > 50 {
		return nil, shared.ErrInvalidInput
	}
	if passwordHash == "" {
		return nil, shared.ErrInvalidInput
	}
	return &User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		displayName:  displayName,
	}, nil
}

// Reconstruct hydrates a User from persistence.
func Reconstruct(id shared.UserID, email, passwordHash, displayName string, verifiedAt *time.Time, createdAt time.Time) *User {
	return &User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		displayName:  displayName,
		verifiedAt:   verifiedAt,
		createdAt:    createdAt,
	}
}

// Accessors
func (u *User) ID() shared.UserID    { return u.id }
func (u *User) Email() string         { return u.email }
func (u *User) PasswordHash() string  { return u.passwordHash }
func (u *User) DisplayName() string   { return u.displayName }
func (u *User) VerifiedAt() *time.Time { return u.verifiedAt }
func (u *User) CreatedAt() time.Time  { return u.createdAt }

// Repository defines persistence for users.
type Repository interface {
	Save(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id shared.UserID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
}
