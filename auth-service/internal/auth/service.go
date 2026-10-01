// Package auth implements the auth-service gRPC API.
package auth

import (
	"context"
	"errors"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
)

// dummyHash is compared against when a username does not exist, so that
// unknown users and wrong passwords take the same time to reject.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

// Service implements authv1.AuthServiceServer.
type Service struct {
	authv1.UnimplementedAuthServiceServer

	db     *gorm.DB
	codes  CodeStore
	mailer Mailer
	tokens *TokenManager
	log    *slog.Logger
}

// NewService returns an auth Service.
func NewService(db *gorm.DB, codes CodeStore, mailer Mailer, tokens *TokenManager, log *slog.Logger) *Service {
	return &Service{db: db, codes: codes, mailer: mailer, tokens: tokens, log: log}
}

// Signup creates an unverified customer account and sends a verification code.
func (s *Service) Signup(ctx context.Context, req *authv1.SignupRequest) (*authv1.SignupResponse, error) {
	email := normalizeEmail(req.GetEmail())
	if err := validateSignup(req.GetUsername(), email, req.GetPassword()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "hash password", err)
	}

	user := User{
		Username:     req.GetUsername(),
		Email:        email,
		Phone:        req.GetPhone(),
		PasswordHash: string(hash),
		Role:         RoleCustomer,
	}
	if a := req.GetAddress(); a != nil {
		user.Address = &Address{
			AddressLine1: a.GetAddressLine1(),
			AddressLine2: a.GetAddressLine2(),
			City:         a.GetCity(),
			State:        a.GetState(),
			PostalCode:   a.GetPostalCode(),
			Country:      a.GetCountry(),
		}
	}

	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, status.Error(codes.AlreadyExists, "username or email is already registered")
		}
		return nil, grpcx.Internal(ctx, s.log, "create user", err)
	}

	if err := s.sendCode(ctx, user.Email); err != nil {
		return nil, grpcx.Internal(ctx, s.log, "send verification code", err)
	}
	return &authv1.SignupResponse{User: toProto(&user)}, nil
}

// VerifyEmail marks the account as verified when the code matches.
func (s *Service) VerifyEmail(ctx context.Context, req *authv1.VerifyEmailRequest) (*authv1.VerifyEmailResponse, error) {
	email := normalizeEmail(req.GetEmail())
	switch err := s.codes.Verify(ctx, email, req.GetCode()); {
	case errors.Is(err, ErrCodeInvalid):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrTooManyAttempts):
		return nil, status.Error(codes.ResourceExhausted, err.Error())
	case err != nil:
		return nil, grpcx.Internal(ctx, s.log, "verify code", err)
	}

	res := s.db.WithContext(ctx).Model(&User{}).Where("email = ?", email).Update("email_verified", true)
	if res.Error != nil {
		return nil, grpcx.Internal(ctx, s.log, "mark email verified", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &authv1.VerifyEmailResponse{}, nil
}

// ResendVerificationCode issues a new code for an unverified account. It
// succeeds silently for unknown or verified emails to avoid account enumeration.
func (s *Service) ResendVerificationCode(ctx context.Context, req *authv1.ResendVerificationCodeRequest) (*authv1.ResendVerificationCodeResponse, error) {
	email := normalizeEmail(req.GetEmail())
	var user User
	err := s.db.WithContext(ctx).Where("email = ? AND email_verified = false", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &authv1.ResendVerificationCodeResponse{}, nil
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "load user", err)
	}
	if err := s.sendCode(ctx, email); err != nil {
		return nil, grpcx.Internal(ctx, s.log, "send verification code", err)
	}
	return &authv1.ResendVerificationCodeResponse{}, nil
}

// Login checks the credentials and returns a signed access token.
func (s *Service) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	var user User
	err := s.db.WithContext(ctx).Preload("Address").Where("username = ?", req.GetUsername()).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcx.Internal(ctx, s.log, "load user", err)
	}

	hash := dummyHash
	if err == nil {
		hash = []byte(user.PasswordHash)
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(req.GetPassword())) != nil || user.ID == 0 {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}
	if !user.EmailVerified {
		return nil, status.Error(codes.FailedPrecondition, "email address is not verified")
	}

	token, exp, err := s.tokens.Issue(user.ID, user.Role)
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "issue token", err)
	}
	return &authv1.LoginResponse{AccessToken: token, ExpiresAt: exp.Unix(), User: toProto(&user)}, nil
}

// ValidateToken returns the identity carried by a valid access token.
func (s *Service) ValidateToken(_ context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	userID, role, err := s.tokens.Validate(req.GetToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}
	return &authv1.ValidateTokenResponse{UserId: userID, Role: role}, nil
}

// GetUser returns a user's profile.
func (s *Service) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	var user User
	err := s.db.WithContext(ctx).Preload("Address").First(&user, req.GetUserId()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "load user", err)
	}
	return &authv1.GetUserResponse{User: toProto(&user)}, nil
}

// SeedAdmin creates the admin account if no user with that username exists.
func SeedAdmin(ctx context.Context, db *gorm.DB, username, email, password string) (bool, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	admin := User{
		Username:      username,
		Email:         normalizeEmail(email),
		PasswordHash:  string(hash),
		Role:          RoleAdmin,
		EmailVerified: true,
	}
	res := db.WithContext(ctx).Where(User{Username: username}).FirstOrCreate(&admin)
	return res.RowsAffected > 0, res.Error
}

func (s *Service) sendCode(ctx context.Context, email string) error {
	code, err := s.codes.Issue(ctx, email)
	if err != nil {
		return err
	}
	return s.mailer.SendVerificationCode(ctx, email, code)
}

func toProto(u *User) *authv1.User {
	out := &authv1.User{
		Id:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		Phone:         u.Phone,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
	}
	if a := u.Address; a != nil {
		out.Address = &authv1.Address{
			AddressLine1: a.AddressLine1,
			AddressLine2: a.AddressLine2,
			City:         a.City,
			State:        a.State,
			PostalCode:   a.PostalCode,
			Country:      a.Country,
		}
	}
	return out
}
