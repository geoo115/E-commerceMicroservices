package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSignup(t *testing.T) {
	tests := []struct {
		name, username, email, password string
		wantErr                         bool
	}{
		{"valid", "alice_1", "alice@example.com", "password123", false},
		{"short username", "al", "alice@example.com", "password123", true},
		{"username with spaces", "alice smith", "alice@example.com", "password123", true},
		{"invalid email", "alice", "not-an-email", "password123", true},
		{"email with display name", "alice", "Alice <alice@example.com>", "password123", true},
		{"short password", "alice", "alice@example.com", "short", true},
		{"password over bcrypt limit", "alice", "alice@example.com", strings.Repeat("a", 73), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSignup(tt.username, tt.email, tt.password)
			assert.Equal(t, tt.wantErr, err != nil, "err = %v", err)
		})
	}
}

func TestNewCode(t *testing.T) {
	seen := map[string]bool{}
	for range 50 {
		code, err := newCode()
		assert.NoError(t, err)
		assert.Regexp(t, `^[0-9]{6}$`, code)
		seen[code] = true
	}
	assert.Greater(t, len(seen), 45, "codes should be random")
}
