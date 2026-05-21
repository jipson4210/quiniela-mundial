package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
	"github.com/josemontalban/quiniela-mundial/internal/infrastructure/email"
)

// InviteMemberInput holds the data to invite someone to a pool.
type InviteMemberInput struct {
	PoolID    string
	Email     string
	InvitedBy string
}

// InviteMemberOutput holds the invitation result.
type InviteMemberOutput struct {
	InvitationID string
	Token        string
	ExpiresAt    string
}

// InviteMember creates an invitation to join a pool and sends an email.
type InviteMember struct {
	pools  pool.Repository
	email  email.Sender
	appURL string // base URL for invitation links
}

func NewInviteMember(pools pool.Repository, emailSender email.Sender, appURL string) *InviteMember {
	return &InviteMember{pools: pools, email: emailSender, appURL: appURL}
}

func (uc *InviteMember) Execute(ctx context.Context, input InviteMemberInput) (*InviteMemberOutput, error) {
	// Verify the pool exists
	_, err := uc.pools.FindByID(ctx, shared.PoolID(input.PoolID))
	if err != nil {
		return nil, fmt.Errorf("invite: pool: %w", err)
	}

	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("invite: generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	invitationID := shared.InvitationID(uuid.Must(uuid.NewV7()).String())
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	inv, err := pool.NewInvitation(
		invitationID,
		shared.PoolID(input.PoolID),
		input.Email,
		token,
		shared.UserID(input.InvitedBy),
		expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("invite: %w", err)
	}

	if err := uc.pools.SaveInvitation(ctx, inv); err != nil {
		return nil, fmt.Errorf("invite: save: %w", err)
	}

	// Send invitation email (noop in dev, real SMTP in prod)
	inviteLink := fmt.Sprintf("%s/invitations/%s/accept", uc.appURL, token)
	_ = uc.email.Send(ctx, input.Email, "Te han invitado a una quiniela",
		fmt.Sprintf("Únete en: %s", inviteLink))

	return &InviteMemberOutput{
		InvitationID: string(invitationID),
		Token:        token,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
	}, nil
}
