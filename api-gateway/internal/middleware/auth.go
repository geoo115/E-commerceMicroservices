// Package middleware contains the HTTP middleware of the API gateway.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
)

const (
	userIDKey = "auth.user_id"
	roleKey   = "auth.role"

	// RoleAdmin is the role allowed to manage the catalog, stock and fulfilment.
	RoleAdmin = "admin"
)

// TokenValidator is the subset of auth-service used to validate tokens.
type TokenValidator interface {
	ValidateToken(ctx context.Context, in *authv1.ValidateTokenRequest, opts ...grpc.CallOption) (*authv1.ValidateTokenResponse, error)
}

// Authenticate requires a valid "Authorization: Bearer <token>" header and
// stores the caller's identity in the request context.
func Authenticate(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			abort(c, http.StatusUnauthorized, "UNAUTHENTICATED", "missing or malformed bearer token")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		resp, err := validator.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
		switch status.Code(err) {
		case codes.OK:
		case codes.Unauthenticated:
			abort(c, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid or expired token")
			return
		default:
			abort(c, http.StatusServiceUnavailable, "UNAVAILABLE", "authentication service unavailable")
			return
		}

		c.Set(userIDKey, resp.GetUserId())
		c.Set(roleKey, resp.GetRole())
		c.Next()
	}
}

// RequireRole allows the request only if the authenticated caller has role.
// It must run after Authenticate.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if Role(c) != role {
			abort(c, http.StatusForbidden, "PERMISSION_DENIED", "insufficient permissions")
			return
		}
		c.Next()
	}
}

// UserID returns the authenticated user's ID (0 if unauthenticated).
func UserID(c *gin.Context) uint64 { return c.GetUint64(userIDKey) }

// Role returns the authenticated user's role.
func Role(c *gin.Context) string { return c.GetString(roleKey) }

// IsAdmin reports whether the caller is an admin.
func IsAdmin(c *gin.Context) bool { return Role(c) == RoleAdmin }

func abort(c *gin.Context, httpStatus int, code, msg string) {
	c.AbortWithStatusJSON(httpStatus, gin.H{"error": gin.H{"code": code, "message": msg}})
}
