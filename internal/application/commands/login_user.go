package commands

import (
	"context"
	"fmt"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

// LoginUserInput holds login credentials.
type LoginUserInput struct {
	Email    string
	Password string
}

// LoginUserOutput holds the result of a successful login.
type LoginUserOutput struct {
	UserID      string
	Email       string
	DisplayName string
}

// LoginUser authenticates a user by email and password.
type LoginUser struct {
	users user.Repository
}

func NewLoginUser(users user.Repository) *LoginUser {
	return &LoginUser{users: users}
}

func (uc *LoginUser) Execute(ctx context.Context, input LoginUserInput) (*LoginUserOutput, error) {
	u, err := uc.users.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", shared.ErrUnauthorized)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash()), []byte(input.Password)); err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", shared.ErrUnauthorized)
	}

	return &LoginUserOutput{
		UserID:      string(u.ID()),
		Email:       u.Email(),
		DisplayName: u.DisplayName(),
	}, nil
}
