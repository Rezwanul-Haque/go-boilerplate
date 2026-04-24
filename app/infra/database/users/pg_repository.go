package users

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	usersFeature "go-boilerplate/app/features/users"
)

type Repository interface {
	usersFeature.UserRepository
	usersFeature.PasswordResetRepository
}

type pgRepository struct {
	db *sql.DB
}

func NewPgRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, user *usersFeature.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *pgRepository) FindByEmail(ctx context.Context, email string) (*usersFeature.User, error) {
	const q = `
		SELECT id, email, password_hash, reset_token, reset_token_expires_at, created_at, updated_at
		FROM users WHERE email = $1`
	return r.scan(r.db.QueryRowContext(ctx, q, email))
}

func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*usersFeature.User, error) {
	const q = `
		SELECT id, email, password_hash, reset_token, reset_token_expires_at, created_at, updated_at
		FROM users WHERE id = $1`
	return r.scan(r.db.QueryRowContext(ctx, q, id))
}

func (r *pgRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	const q = `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, q, hashedPassword, time.Now(), id)
	return err
}

func (r *pgRepository) SaveResetToken(ctx context.Context, id uuid.UUID, tok string, expiresAt time.Time) error {
	const q = `UPDATE users SET reset_token = $1, reset_token_expires_at = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, q, tok, expiresAt, time.Now(), id)
	return err
}

func (r *pgRepository) FindByResetToken(ctx context.Context, tok string) (*usersFeature.User, error) {
	const q = `
		SELECT id, email, password_hash, reset_token, reset_token_expires_at, created_at, updated_at
		FROM users WHERE reset_token = $1`
	return r.scan(r.db.QueryRowContext(ctx, q, tok))
}

func (r *pgRepository) ClearResetToken(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE users SET reset_token = NULL, reset_token_expires_at = NULL, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, time.Now(), id)
	return err
}

func (r *pgRepository) scan(row *sql.Row) (*usersFeature.User, error) {
	u := &usersFeature.User{}
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash,
		&u.ResetToken, &u.ResetTokenExpiresAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, usersFeature.ErrUserNotFound
	}
	return u, err
}
