package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/geoo115/E-commerceMicroservices/api-gateway/internal/middleware"
	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
)

type signupRequest struct {
	Username string      `json:"username" binding:"required"`
	Email    string      `json:"email" binding:"required,email"`
	Password string      `json:"password" binding:"required,min=8,max=72"`
	Phone    string      `json:"phone" binding:"omitempty,max=32"`
	Address  *addressDTO `json:"address"`
}

// Signup registers a customer account. POST /api/v1/auth/signup
func (h *Handler) Signup(c *gin.Context) {
	var req signupRequest
	if !bindJSON(c, &req) {
		return
	}
	in := &authv1.SignupRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Phone:    req.Phone,
	}
	if a := req.Address; a != nil {
		in.Address = &authv1.Address{
			AddressLine1: a.AddressLine1,
			AddressLine2: a.AddressLine2,
			City:         a.City,
			State:        a.State,
			PostalCode:   a.PostalCode,
			Country:      a.Country,
		}
	}
	resp, err := h.clients.Auth.Signup(c.Request.Context(), in)
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"user":    toUser(resp.GetUser()),
		"message": "account created; check your email for the verification code",
	})
}

type verifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6,numeric"`
}

// VerifyEmail confirms an email address. POST /api/v1/auth/verify-email
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if !bindJSON(c, &req) {
		return
	}
	_, err := h.clients.Auth.VerifyEmail(c.Request.Context(), &authv1.VerifyEmailRequest{Email: req.Email, Code: req.Code})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "email verified"})
}

type resendCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResendVerificationCode sends a new code. POST /api/v1/auth/resend-code
func (h *Handler) ResendVerificationCode(c *gin.Context) {
	var req resendCodeRequest
	if !bindJSON(c, &req) {
		return
	}
	_, err := h.clients.Auth.ResendVerificationCode(c.Request.Context(), &authv1.ResendVerificationCodeRequest{Email: req.Email})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "if the account exists and is unverified, a new code was sent"})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login exchanges credentials for an access token. POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.clients.Auth.Login(c.Request.Context(), &authv1.LoginRequest{Username: req.Username, Password: req.Password})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.GetAccessToken(),
		"token_type":   "Bearer",
		"expires_at":   time.Unix(resp.GetExpiresAt(), 0).UTC(),
		"user":         toUser(resp.GetUser()),
	})
}

// Me returns the authenticated user's profile. GET /api/v1/users/me
func (h *Handler) Me(c *gin.Context) {
	resp, err := h.clients.Auth.GetUser(c.Request.Context(), &authv1.GetUserRequest{UserId: middleware.UserID(c)})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUser(resp.GetUser()))
}
