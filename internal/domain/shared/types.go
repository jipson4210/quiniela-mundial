// Package shared contains domain primitives used across all bounded contexts:
// strongly-typed IDs, common errors, and base value objects.
package shared

import "errors"

// ID types — string-backed for UUID v7 interoperability
type (
	UserID         string
	PoolID         string
	TournamentID   string
	TeamID         string
	MatchID        string
	GroupID        string
	StageID        string
	PredictionID   string
	BracketPredID  string
	ScoreEntryID   string
	InvitationID   string
)

// Common domain errors
var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrInvalidInput   = errors.New("invalid input")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrConflict       = errors.New("conflict")
)
