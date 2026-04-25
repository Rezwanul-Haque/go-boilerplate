package users_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"go-boilerplate/app/features/users"
	"go-boilerplate/app/shared/model"
	"go-boilerplate/app/shared/token"
)

// --- mock UserRepository ---

type mockUserRepo struct {
	byEmail   map[string]*users.User
	byID      map[uuid.UUID]*users.User
	createErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byEmail: make(map[string]*users.User),
		byID:    make(map[uuid.UUID]*users.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, u *users.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.byEmail[u.Email] = u
	m.byID[u.ID] = u
	return nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*users.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*users.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockUserRepo) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if u, ok := m.byID[id]; ok {
		u.PasswordHash = hash
	}
	return nil
}

func (m *mockUserRepo) List(_ context.Context, limit, offset int) ([]*users.User, int64, error) {
	all := make([]*users.User, 0, len(m.byID))
	for _, u := range m.byID {
		all = append(all, u)
	}
	total := int64(len(all))
	if offset >= len(all) {
		return []*users.User{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (m *mockUserRepo) ListAfterCursor(_ context.Context, cursor time.Time, limit int) ([]*users.User, error) {
	var result []*users.User
	for _, u := range m.byID {
		if cursor.IsZero() || u.CreatedAt.Before(cursor) {
			result = append(result, u)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// --- mock PasswordResetRepository ---

type mockResetRepo struct {
	tokens  map[string]*users.User
	saveErr error
}

func newMockResetRepo() *mockResetRepo {
	return &mockResetRepo{tokens: make(map[string]*users.User)}
}

func (m *mockResetRepo) SaveResetToken(_ context.Context, id uuid.UUID, tok string, _ time.Time) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.tokens[tok] = &users.User{Base: model.Base{ID: id}}
	return nil
}

func (m *mockResetRepo) FindByResetToken(_ context.Context, tok string) (*users.User, error) {
	u, ok := m.tokens[tok]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockResetRepo) ClearResetToken(_ context.Context, id uuid.UUID) error {
	for k, u := range m.tokens {
		if u.ID == id {
			delete(m.tokens, k)
		}
	}
	return nil
}

// --- mock Notifier ---

type mockNotifier struct {
	called bool
	email  string
}

func (m *mockNotifier) SendPasswordReset(_ context.Context, email, _ string) error {
	m.called = true
	m.email = email
	return nil
}

// --- helpers ---

const testJWTSecret = "supersecretkey1234567890abcdefghij"

func newTestService(userRepo *mockUserRepo, resetRepo *mockResetRepo, notifier *mockNotifier) users.Service {
	maker := token.NewJWTMaker(testJWTSecret)
	return users.NewService(userRepo, resetRepo, notifier, maker)
}

func hashedPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

// --- tests ---

func TestSignup_Success(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	resp, err := svc.Signup(context.Background(), users.SignupRequest{
		Email: "new@example.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "new@example.com", resp.User.Email)
}

func TestSignup_EmailAlreadyExists(t *testing.T) {
	repo := newMockUserRepo()
	repo.byEmail["exists@example.com"] = &users.User{Base: model.Base{ID: uuid.New()}, Email: "exists@example.com"}
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	_, err := svc.Signup(context.Background(), users.SignupRequest{
		Email: "exists@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, users.ErrEmailAlreadyExists)
}

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	u := &users.User{Base: model.Base{ID: id}, Email: "login@example.com", PasswordHash: hashedPassword(t, "password123")}
	repo.byEmail[u.Email] = u
	repo.byID[id] = u
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	resp, err := svc.Login(context.Background(), users.LoginRequest{
		Email: "login@example.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	u := &users.User{Base: model.Base{ID: id}, Email: "login@example.com", PasswordHash: hashedPassword(t, "password123")}
	repo.byEmail[u.Email] = u
	repo.byID[id] = u
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	_, err := svc.Login(context.Background(), users.LoginRequest{
		Email: "login@example.com", Password: "wrongpassword",
	})

	assert.ErrorIs(t, err, users.ErrInvalidCredentials)
}

func TestLogin_EmailNotFound(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	_, err := svc.Login(context.Background(), users.LoginRequest{
		Email: "nobody@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, users.ErrInvalidCredentials)
}

func TestForgotPassword_EmailNotFound_ReturnsNil(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	err := svc.ForgotPassword(context.Background(), users.ForgotPasswordRequest{
		Email: "nobody@example.com",
	})

	assert.NoError(t, err)
}

func TestForgotPassword_Success_CallsNotifier(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	u := &users.User{Base: model.Base{ID: id}, Email: "forgot@example.com"}
	repo.byEmail[u.Email] = u
	repo.byID[id] = u
	notifier := &mockNotifier{}
	svc := newTestService(repo, newMockResetRepo(), notifier)

	err := svc.ForgotPassword(context.Background(), users.ForgotPasswordRequest{
		Email: "forgot@example.com",
	})

	require.NoError(t, err)
	assert.True(t, notifier.called)
	assert.Equal(t, "forgot@example.com", notifier.email)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	err := svc.ResetPassword(context.Background(), users.ResetPasswordRequest{
		Token: "badtoken", Password: "newpassword",
	})

	assert.ErrorIs(t, err, users.ErrInvalidResetToken)
}


func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	u := &users.User{Base: model.Base{ID: id}, Email: "change@example.com", PasswordHash: hashedPassword(t, "currentpass")}
	repo.byID[id] = u
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	err := svc.ChangePassword(context.Background(), id, users.ChangePasswordRequest{
		CurrentPassword: "wrongpass", NewPassword: "newpassword",
	})

	assert.ErrorIs(t, err, users.ErrWrongPassword)
}

func TestListUsers_ReturnsPaginatedResults(t *testing.T) {
	repo := newMockUserRepo()
	for i := range 5 {
		id := uuid.New()
		u := &users.User{Base: model.Base{ID: id, CreatedAt: time.Now()}, Email: fmt.Sprintf("user%d@example.com", i)}
		repo.byID[id] = u
	}
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	resp, err := svc.ListUsers(context.Background(), 1, 3)

	require.NoError(t, err)
	assert.LessOrEqual(t, len(resp.Data), 3)
	assert.Equal(t, int64(5), resp.Total)
	assert.Equal(t, 2, resp.TotalPages)
}

func TestListUsersCursor_FirstPage_EmptyCursor(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	repo.byID[id] = &users.User{Base: model.Base{ID: id, CreatedAt: time.Now()}, Email: "a@example.com"}
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	resp, err := svc.ListUsersCursor(context.Background(), "", 10)

	require.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	assert.False(t, resp.HasMore)
	assert.Empty(t, resp.NextCursor)
}

func TestListUsersCursor_InvalidCursor_ReturnsError(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	_, err := svc.ListUsersCursor(context.Background(), "notbase64!!", 10)

	assert.Error(t, err)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	_, err := svc.RefreshToken(context.Background(), users.RefreshTokenRequest{
		RefreshToken: "notavalidtoken",
	})

	assert.ErrorIs(t, err, users.ErrInvalidRefreshToken)
}

func TestRefreshToken_AccessTokenUsedAsRefresh(t *testing.T) {
	maker := token.NewJWTMaker(testJWTSecret)
	accessTok, err := maker.CreateToken(uuid.New(), "r@example.com", token.AccessToken, time.Minute, "")
	require.NoError(t, err)

	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})
	_, err = svc.RefreshToken(context.Background(), users.RefreshTokenRequest{
		RefreshToken: accessTok,
	})

	assert.ErrorIs(t, err, users.ErrInvalidRefreshToken)
}
