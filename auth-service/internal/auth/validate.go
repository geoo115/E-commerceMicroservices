package auth

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// MinPasswordLength is the minimum accepted password length.
const MinPasswordLength = 8

func validateSignup(username, email, password string) error {
	if !usernamePattern.MatchString(username) {
		return errors.New("username must be 3-32 characters: letters, digits or underscore")
	}
	if !validEmail(email) {
		return errors.New("email is not a valid address")
	}
	if len(password) < MinPasswordLength {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 72 { // bcrypt only uses the first 72 bytes
		return errors.New("password must be at most 72 characters")
	}
	return nil
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
