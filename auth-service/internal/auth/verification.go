package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	codeTTL         = 15 * time.Minute
	maxCodeAttempts = 5
)

var (
	// ErrCodeInvalid is returned for a wrong, expired or missing code.
	ErrCodeInvalid = errors.New("invalid or expired verification code")
	// ErrTooManyAttempts is returned once maxCodeAttempts wrong codes were submitted.
	ErrTooManyAttempts = errors.New("too many attempts, request a new verification code")
)

// CodeStore stores email verification codes.
type CodeStore interface {
	// Issue generates, stores and returns a fresh code for email.
	Issue(ctx context.Context, email string) (string, error)
	// Verify checks code for email and consumes it on success.
	Verify(ctx context.Context, email, code string) error
}

// RedisCodeStore keeps codes in Redis with a TTL and limits guessing attempts.
type RedisCodeStore struct {
	rdb *redis.Client
}

// NewRedisCodeStore returns a CodeStore backed by rdb.
func NewRedisCodeStore(rdb *redis.Client) *RedisCodeStore { return &RedisCodeStore{rdb: rdb} }

func codeKey(email string) string     { return "verify:" + email }
func attemptsKey(email string) string { return "verify:attempts:" + email }

// Issue implements CodeStore.
func (s *RedisCodeStore) Issue(ctx context.Context, email string) (string, error) {
	code, err := newCode()
	if err != nil {
		return "", err
	}
	_, err = s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, codeKey(email), code, codeTTL)
		p.Del(ctx, attemptsKey(email))
		return nil
	})
	return code, err
}

// Verify implements CodeStore.
func (s *RedisCodeStore) Verify(ctx context.Context, email, code string) error {
	attempts, err := s.rdb.Incr(ctx, attemptsKey(email)).Result()
	if err != nil {
		return err
	}
	s.rdb.Expire(ctx, attemptsKey(email), codeTTL)
	if attempts > maxCodeAttempts {
		s.rdb.Del(ctx, codeKey(email))
		return ErrTooManyAttempts
	}

	stored, err := s.rdb.Get(ctx, codeKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrCodeInvalid
	}
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(code)) != 1 {
		return ErrCodeInvalid
	}
	return s.rdb.Del(ctx, codeKey(email), attemptsKey(email)).Err()
}

// newCode returns a cryptographically random 6-digit code.
func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
