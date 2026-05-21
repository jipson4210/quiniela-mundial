package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

// RegisterUserInput holds the data needed to register.
type RegisterUserInput struct {
	Email       string
	Password    string
	DisplayName string
}

// RegisterUserOutput holds the result of registration.
type RegisterUserOutput struct {
	UserID      string
	Email       string
	DisplayName string
}

// RegisterUser creates a new user account.
type RegisterUser struct {
	users user.Repository
}

func NewRegisterUser(users user.Repository) *RegisterUser {
	return &RegisterUser{users: users}
}

func (uc *RegisterUser) Execute(ctx context.Context, input RegisterUserInput) (*RegisterUserOutput, error) {
	// Check for duplicate email
	existing, err := uc.users.FindByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("%w: email already registered", shared.ErrAlreadyExists)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("register: hash password: %w", err)
	}

	u, err := user.New(
		shared.UserID(uuid.Must(uuid.NewV7()).String()),
		input.Email,
		string(hash),
		input.DisplayName,
	)
	if err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	if err := uc.users.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("register: save: %w", err)
	}

	return &RegisterUserOutput{
		UserID:      string(u.ID()),
		Email:       u.Email(),
		DisplayName: u.DisplayName(),
	}, nil
}
