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
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1`
	return r.scan(r.db.QueryRowContext(ctx, q, email))
}

func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*usersFeature.User, error) {
	const q = `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE id = $1`
	return r.scan(r.db.QueryRowContext(ctx, q, id))
}

func (r *pgRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	const q = `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, q, hashedPassword, time.Now(), id)
	return err
}

func (r *pgRepository) List(ctx context.Context, limit, offset int) ([]*usersFeature.User, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return r.scanRows(rows, total)
}

func (r *pgRepository) ListAfterCursor(ctx context.Context, cursor time.Time, limit int) ([]*usersFeature.User, error) {
	var (
		q    string
		args []any
	)
	if cursor.IsZero() {
		q = `SELECT id, email, password_hash, created_at, updated_at
		     FROM users ORDER BY created_at DESC, id DESC LIMIT $1`
		args = []any{limit}
	} else {
		q = `SELECT id, email, password_hash, created_at, updated_at
		     FROM users WHERE created_at < $1 ORDER BY created_at DESC, id DESC LIMIT $2`
		args = []any{cursor, limit}
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users, _, err := r.scanRows(rows, 0)
	return users, err
}

func (r *pgRepository) scanRows(rows *sql.Rows, knownTotal int64) ([]*usersFeature.User, int64, error) {
	var users []*usersFeature.User
	for rows.Next() {
		u := &usersFeature.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, knownTotal, rows.Err()
}

func (r *pgRepository) scan(row *sql.Row) (*usersFeature.User, error) {
	u := &usersFeature.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, usersFeature.ErrUserNotFound
	}
	return u, err
}
