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
	"go-boilerplate/app/shared/model"
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
		Base:         model.Base{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Email:        "create@example.com",
		PasswordHash: "hashedpassword",
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
		Base:         model.Base{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Email:        "dup@example.com",
		PasswordHash: "hash",
	}

	require.NoError(s.T(), s.repo.Create(ctx, user))
	err := s.repo.Create(ctx, &usersFeature.User{
		Base:         model.Base{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Email:        "dup@example.com",
		PasswordHash: "hash2",
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
		Base:         model.Base{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Email:        "update@example.com",
		PasswordHash: "oldhash",
	}
	require.NoError(s.T(), s.repo.Create(ctx, user))

	found, err := s.repo.FindByEmail(ctx, "update@example.com")
	require.NoError(s.T(), err)

	require.NoError(s.T(), s.repo.UpdatePassword(ctx, found.ID, "newhash"))

	updated, err := s.repo.FindByID(ctx, found.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "newhash", updated.PasswordHash)
}

func TestPgRepositorySuite(t *testing.T) {
	suite.Run(t, new(PgRepositorySuite))
}
