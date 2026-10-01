package auth

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/geoo115/E-commerceMicroservices/pkg/database/dbtest"
	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
)

// memoryCodes is an in-memory CodeStore that remembers the last issued code.
type memoryCodes struct{ codes map[string]string }

func (m *memoryCodes) Issue(_ context.Context, email string) (string, error) {
	m.codes[email] = "123456"
	return "123456", nil
}

func (m *memoryCodes) Verify(_ context.Context, email, code string) error {
	if m.codes[email] != code {
		return ErrCodeInvalid
	}
	delete(m.codes, email)
	return nil
}

func newService(t *testing.T) *Service {
	tokens, err := NewTokenManager(testSecret, time.Hour)
	require.NoError(t, err)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(dbtest.New(t, Models...), &memoryCodes{codes: map[string]string{}}, LogMailer{Log: log}, tokens, log)
}

func TestSignupVerifyLogin(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	_, err := s.Signup(ctx, &authv1.SignupRequest{Username: "alice", Email: "Alice@Example.com", Password: "password123"})
	require.NoError(t, err)

	_, err = s.Signup(ctx, &authv1.SignupRequest{Username: "alice", Email: "other@example.com", Password: "password123"})
	assert.Equal(t, codes.AlreadyExists, status.Code(err), "duplicate username")

	_, err = s.Login(ctx, &authv1.LoginRequest{Username: "alice", Password: "password123"})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err), "email not verified yet")

	_, err = s.VerifyEmail(ctx, &authv1.VerifyEmailRequest{Email: "alice@example.com", Code: "000000"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = s.VerifyEmail(ctx, &authv1.VerifyEmailRequest{Email: "alice@example.com", Code: "123456"})
	require.NoError(t, err)

	resp, err := s.Login(ctx, &authv1.LoginRequest{Username: "alice", Password: "password123"})
	require.NoError(t, err)
	assert.Equal(t, RoleCustomer, resp.GetUser().GetRole())

	valid, err := s.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: resp.GetAccessToken()})
	require.NoError(t, err)
	assert.Equal(t, resp.GetUser().GetId(), valid.GetUserId())
}

func TestLogin_SameErrorForUnknownUserAndWrongPassword(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	_, err := SeedAdmin(ctx, s.db, "admin", "admin@example.com", "admin-password")
	require.NoError(t, err)

	_, errWrongPassword := s.Login(ctx, &authv1.LoginRequest{Username: "admin", Password: "nope"})
	_, errUnknownUser := s.Login(ctx, &authv1.LoginRequest{Username: "ghost", Password: "nope"})

	assert.Equal(t, codes.Unauthenticated, status.Code(errWrongPassword))
	assert.Equal(t, status.Convert(errWrongPassword).Message(), status.Convert(errUnknownUser).Message(),
		"responses must not reveal whether the username exists")
}

func TestSeedAdmin_IsIdempotent(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	created, err := SeedAdmin(ctx, s.db, "admin", "admin@example.com", "admin-password")
	require.NoError(t, err)
	assert.True(t, created)
	created, err = SeedAdmin(ctx, s.db, "admin", "admin@example.com", "admin-password")
	require.NoError(t, err)
	assert.False(t, created)
}
