package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenIssuer = "auth-service"

// MinSecretLength is the minimum accepted HMAC secret length in bytes.
const MinSecretLength = 32

// Claims are the JWT claims issued by auth-service.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager issues and validates HS256 access tokens.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokenManager returns a TokenManager. The secret must be at least MinSecretLength bytes.
func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	if len(secret) < MinSecretLength {
		return nil, fmt.Errorf("JWT secret must be at least %d bytes", MinSecretLength)
	}
	return &TokenManager{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

// Issue returns a signed token for the user and its expiry time.
func (m *TokenManager) Issue(userID uint64, role string) (string, time.Time, error) {
	now := m.now()
	exp := now.Add(m.ttl)
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(userID, 10),
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return signed, exp, err
}

// Validate parses token and returns the user ID and role it was issued for.
func (m *TokenManager) Validate(token string) (uint64, string, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return 0, "", err
	}
	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || userID == 0 {
		return 0, "", errors.New("invalid subject claim")
	}
	return userID, claims.Role, nil
}
