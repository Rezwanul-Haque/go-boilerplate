//go:build integration

package users_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/google/uuid"

	usersFeature "go-boilerplate/app/features/users"
	dbUsers "go-boilerplate/app/infra/database/users"
)

func integrationDSN() string {
	if dsn := os.Getenv("TEST_DB_DSN"); dsn != "" {
		return dsn
	}
	return "host=localhost port=5432 user=postgres password=postgres dbname=go_boilerplate sslmode=disable"
}

type PgRepositorySuite struct {
	suite.Suite
	db   *sql.DB
	repo dbUsers.Repository
}

func (s *PgRepositorySuite) SetupSuite() {
	db, err := sql.Open("pgx", integrationDSN())
	require.NoError(s.T(), err)
	require.NoError(s.T(), db.PingContext(context.Background()))
	s.db = db
	s.repo = dbUsers.NewPgRepository(db)
}

func (s *PgRepositorySuite) TearDownSuite() {
	s.db.Close()
}

func (s *PgRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(context.Background(), "DELETE FROM users")
	require.NoError(s.T(), err)
}

func (s *PgRepositorySuite) TestCreate_AndFindByEmail() {
	ctx := context.Background()
	user := &usersFeature.User{
		Email:        "create@example.com",
		PasswordHash: "hashedpassword",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := s.repo.Create(ctx, user)
	require.NoError(s.T(), err)

	found, err := s.repo.FindByEmail(ctx, "create@example.com")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "create@example.com", found.Email)
	assert.Equal(s.T(), "hashedpassword", found.PasswordHash)
}

func (s *PgRepositorySuite) TestCreate_DuplicateEmail_ReturnsError() {
	ctx := context.Background()
	user := &usersFeature.User{
		Email:        "dup@example.com",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	require.NoError(s.T(), s.repo.Create(ctx, user))
	err := s.repo.Create(ctx, &usersFeature.User{
		Email:        "dup@example.com",
		PasswordHash: "hash2",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	assert.Error(s.T(), err)
}

func (s *PgRepositorySuite) TestFindByEmail_NotFound() {
	_, err := s.repo.FindByEmail(context.Background(), "nobody@example.com")
	assert.ErrorIs(s.T(), err, usersFeature.ErrUserNotFound)
}

func (s *PgRepositorySuite) TestFindByID_NotFound() {
	_, err := s.repo.FindByID(context.Background(), uuid.UUID{})
	assert.ErrorIs(s.T(), err, usersFeature.ErrUserNotFound)
}

func (s *PgRepositorySuite) TestUpdatePassword() {
	ctx := context.Background()
	user := &usersFeature.User{
		Email:        "update@example.com",
		PasswordHash: "oldhash",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(s.T(), s.repo.Create(ctx, user))

	found, err := s.repo.FindByEmail(ctx, "update@example.com")
	require.NoError(s.T(), err)

	require.NoError(s.T(), s.repo.UpdatePassword(ctx, found.ID, "newhash"))

	updated, err := s.repo.FindByID(ctx, found.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "newhash", updated.PasswordHash)
}

func (s *PgRepositorySuite) TestResetToken_SaveFindClear() {
	ctx := context.Background()
	user := &usersFeature.User{
		Email:        "reset@example.com",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(s.T(), s.repo.Create(ctx, user))

	found, err := s.repo.FindByEmail(ctx, "reset@example.com")
	require.NoError(s.T(), err)

	expiresAt := time.Now().Add(time.Hour)
	require.NoError(s.T(), s.repo.SaveResetToken(ctx, found.ID, "myresettoken", expiresAt))

	byToken, err := s.repo.FindByResetToken(ctx, "myresettoken")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), found.ID, byToken.ID)
	assert.NotNil(s.T(), byToken.ResetTokenExpiresAt)

	require.NoError(s.T(), s.repo.ClearResetToken(ctx, found.ID))

	_, err = s.repo.FindByResetToken(ctx, "myresettoken")
	assert.ErrorIs(s.T(), err, usersFeature.ErrUserNotFound)
}

func TestPgRepositorySuite(t *testing.T) {
	suite.Run(t, new(PgRepositorySuite))
}
