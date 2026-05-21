package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/josemontalban/quiniela-mundial/internal/domain/pool"
	"github.com/josemontalban/quiniela-mundial/internal/domain/shared"
)

// AcceptInvitationInput holds the data to accept an invitation.
type AcceptInvitationInput struct {
	Token  string
	UserID string
}

// AcceptInvitationOutput holds the result of accepting.
type AcceptInvitationOutput struct {
	PoolID string
}

// AcceptInvitation redeems an invitation token and adds the user to the pool.
type AcceptInvitation struct {
	pools pool.Repository
}

func NewAcceptInvitation(pools pool.Repository) *AcceptInvitation {
	return &AcceptInvitation{pools: pools}
}

func (uc *AcceptInvitation) Execute(ctx context.Context, input AcceptInvitationInput) (*AcceptInvitationOutput, error) {
	inv, err := uc.pools.FindInvitationByToken(ctx, input.Token)
	if err != nil {
		return nil, fmt.Errorf("accept_invitation: %w", shared.ErrNotFound)
	}

	if inv.IsAccepted() {
		return nil, fmt.Errorf("%w: invitation already accepted", shared.ErrConflict)
	}

	if inv.IsExpired(time.Now()) {
		return nil, fmt.Errorf("%w: invitation expired", shared.ErrInvalidInput)
	}

	// Add user to pool as member
	invitedBy := inv.InvitedBy()
	pm := pool.NewPoolMember(inv.PoolID(), shared.UserID(input.UserID), pool.RoleMember, &invitedBy)
	if err := uc.pools.AddMember(ctx, pm); err != nil {
		return nil, fmt.Errorf("accept_invitation: add member: %w", err)
	}

	if err := uc.pools.AcceptInvitation(ctx, inv.ID(), time.Now()); err != nil {
		return nil, fmt.Errorf("accept_invitation: mark accepted: %w", err)
	}

	return &AcceptInvitationOutput{
		PoolID: string(inv.PoolID()),
	}, nil
}
