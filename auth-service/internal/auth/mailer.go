package auth

import (
	"context"
	"log/slog"
)

// Mailer delivers verification codes to users.
type Mailer interface {
	SendVerificationCode(ctx context.Context, email, code string) error
}

// LogMailer "sends" emails by logging them. It stands in for an SMTP or
// transactional-email provider in local and demo environments.
type LogMailer struct {
	Log *slog.Logger
}

// SendVerificationCode implements Mailer.
func (m LogMailer) SendVerificationCode(ctx context.Context, email, code string) error {
	m.Log.InfoContext(ctx, "verification email (not sent: LogMailer)", "to", email, "code", code)
	return nil
}
