package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestNewTokenManager_RejectsShortSecret(t *testing.T) {
	_, err := NewTokenManager("too-short", time.Hour)
	assert.Error(t, err)
}

func TestTokenManager_RoundTrip(t *testing.T) {
	m, err := NewTokenManager(testSecret, time.Hour)
	require.NoError(t, err)

	token, exp, err := m.Issue(42, RoleAdmin)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), exp, time.Second)

	userID, role, err := m.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), userID)
	assert.Equal(t, RoleAdmin, role)
}

func TestTokenManager_RejectsInvalidTokens(t *testing.T) {
	m, err := NewTokenManager(testSecret, time.Hour)
	require.NoError(t, err)
	valid, _, err := m.Issue(1, RoleCustomer)
	require.NoError(t, err)

	expired := func() string {
		old, _ := NewTokenManager(testSecret, time.Hour)
		old.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
		tok, _, _ := old.Issue(1, RoleCustomer)
		return tok
	}()
	otherSecret := func() string {
		other, _ := NewTokenManager(strings.Repeat("x", 32), time.Hour)
		tok, _, _ := other.Issue(1, RoleAdmin)
		return tok
	}()
	unsigned := func() string {
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
			Role:             RoleAdmin,
			RegisteredClaims: jwt.RegisteredClaims{Subject: "1", Issuer: tokenIssuer},
		}).SignedString(jwt.UnsafeAllowNoneSignatureType)
		return tok
	}()

	tests := map[string]string{
		"expired":      expired,
		"wrong secret": otherSecret,
		"alg none":     unsigned,
		"tampered":     valid[:len(valid)-2] + "xx",
		"garbage":      "not-a-jwt",
		"empty":        "",
	}
	for name, tok := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := m.Validate(tok)
			assert.Error(t, err)
		})
	}
}
