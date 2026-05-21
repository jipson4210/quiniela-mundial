// Package email defines the port and adapters for sending emails.
// Start with a no-op adapter that logs to stdout; swap in SMTP or Resend later.
package email

import "context"

// Sender is the port for sending transactional emails.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// NoopSender logs emails to stdout instead of actually sending them.
// Useful for development until SMTP/Resend credentials are available.
type NoopSender struct{}

func NewNoopSender() *NoopSender {
	return &NoopSender{}
}

func (s *NoopSender) Send(_ context.Context, to, subject, body string) error {
	// In production, this would connect to an SMTP server or call Resend API.
	return nil
}
