package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"go-boilerplate/app/shared/apperror"
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
	ListUsers(ctx context.Context, page, limit int) (*OffsetPageResponse, error)
	ListUsersCursor(ctx context.Context, cursor string, limit int) (*CursorPageResponse, error)
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
	unverified, err := s.token.ParseUnverifiedClaims(req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.repo.FindByID(ctx, unverified.UserID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	claims, err := s.token.VerifyToken(req.RefreshToken, user.PasswordHash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if claims.Type != token.RefreshToken {
		return nil, ErrInvalidRefreshToken
	}

	return s.buildAuthResponse(user)
}

func (s *service) buildAuthResponse(user *User) (*AuthResponse, error) {
	accessTok, err := s.token.CreateToken(user.ID, user.Email, token.AccessToken, 15*time.Minute, user.PasswordHash)
	if err != nil {
		return nil, err
	}

	refreshTok, err := s.token.CreateToken(user.ID, user.Email, token.RefreshToken, 7*24*time.Hour, user.PasswordHash)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessTok,
		RefreshToken: refreshTok,
		User:         UserResponse{ID: user.ID.String(), Email: user.Email},
	}, nil
}

func (s *service) ListUsers(ctx context.Context, page, limit int) (*OffsetPageResponse, error) {
	users, total, err := s.repo.List(ctx, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	data := make([]UserResponse, len(users))
	for i, u := range users {
		data[i] = UserResponse{ID: u.ID.String(), Email: u.Email}
	}

	return &OffsetPageResponse{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *service) ListUsersCursor(ctx context.Context, cursorStr string, limit int) (*CursorPageResponse, error) {
	var cursor time.Time
	if cursorStr != "" {
		decoded, err := base64.URLEncoding.DecodeString(cursorStr)
		if err != nil {
			return nil, apperror.New(http.StatusBadRequest, "invalid cursor")
		}
		cursor, err = time.Parse(time.RFC3339Nano, string(decoded))
		if err != nil {
			return nil, apperror.New(http.StatusBadRequest, "invalid cursor")
		}
	}

	// fetch one extra to detect hasMore
	users, err := s.repo.ListAfterCursor(ctx, cursor, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
	}

	var nextCursor string
	if hasMore && len(users) > 0 {
		last := users[len(users)-1]
		nextCursor = base64.URLEncoding.EncodeToString([]byte(last.CreatedAt.UTC().Format(time.RFC3339Nano)))
	}

	data := make([]UserResponse, len(users))
	for i, u := range users {
		data[i] = UserResponse{ID: u.ID.String(), Email: u.Email}
	}

	return &CursorPageResponse{
		Data:       data,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
