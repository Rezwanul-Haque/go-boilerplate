package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"go-boilerplate/app/shared/ports"
	"go-boilerplate/app/shared/token"
)

type Service interface {
	Signup(ctx context.Context, req SignupRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
	ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error
	RefreshToken(ctx context.Context, req RefreshTokenRequest) (*AuthResponse, error)
}

type service struct {
	repo      UserRepository
	resetRepo PasswordResetRepository
	notifier  ports.Notifier
	token     token.Maker
}

func NewService(repo UserRepository, resetRepo PasswordResetRepository, notifier ports.Notifier, tokenMaker token.Maker) Service {
	return &service{
		repo:      repo,
		resetRepo: resetRepo,
		notifier:  notifier,
		token:     tokenMaker,
	}
}

func (s *service) Signup(ctx context.Context, req SignupRequest) (*AuthResponse, error) {
	existing, err := s.repo.FindByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.buildAuthResponse(user)
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.buildAuthResponse(user)
}

func (s *service) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	resetToken, err := generateSecureToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Hour)
	if err := s.resetRepo.SaveResetToken(ctx, user.ID, resetToken, expiresAt); err != nil {
		return err
	}

	return s.notifier.SendPasswordReset(ctx, user.Email, resetToken)
}

func (s *service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	user, err := s.resetRepo.FindByResetToken(ctx, req.Token)
	if err != nil {
		return ErrInvalidResetToken
	}

	if user.ResetTokenExpiresAt == nil || time.Now().After(*user.ResetTokenExpiresAt) {
		return ErrInvalidResetToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(ctx, user.ID, string(hash)); err != nil {
		return err
	}

	return s.resetRepo.ClearResetToken(ctx, user.ID)
}

func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return ErrWrongPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, userID, string(hash))
}

func (s *service) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*AuthResponse, error) {
	claims, err := s.token.VerifyToken(req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if claims.Type != token.RefreshToken {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.repo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return s.buildAuthResponse(user)
}

func (s *service) buildAuthResponse(user *User) (*AuthResponse, error) {
	accessTok, err := s.token.CreateToken(user.ID, user.Email, token.AccessToken, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshTok, err := s.token.CreateToken(user.ID, user.Email, token.RefreshToken, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessTok,
		RefreshToken: refreshTok,
		User:         UserResponse{ID: user.ID.String(), Email: user.Email},
	}, nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
