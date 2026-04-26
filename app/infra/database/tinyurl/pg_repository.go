package tinyurldb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	feat "go-boilerplate/app/features/tinyurl"
)

type pgRepository struct {
	db *sql.DB
}

func NewPgRepository(db *sql.DB) feat.Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, item *feat.Tinyurl) error {
	const q = `INSERT INTO tinyurl (id, short_code, original_url, click_count, expires_at, created_at, updated_at) 
				VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(
		ctx,
		q,
		item.ID,
		item.ShortCode,
		item.OriginalURL,
		item.ClickCount,
		item.ExpiresAt,
		item.CreatedAt,
		item.UpdatedAt,
	)

	return err
}

func (r *pgRepository) List(ctx context.Context, limit, offset int) ([]*feat.Tinyurl, error) {
	const q = `SELECT id, short_code, original_url, click_count, expires_at, created_at, updated_at 
		FROM tinyurl 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*feat.Tinyurl
	for rows.Next() {
		item := &feat.Tinyurl{}
		if err := rows.Scan(
			&item.ID,
			&item.ShortCode,
			&item.OriginalURL,
			&item.ClickCount,
			&item.ExpiresAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *pgRepository) FindByShortCode(ctx context.Context, shortCode string) (*feat.Tinyurl, error) {
	const q = `SELECT id, short_code, original_url, click_count, expires_at, created_at, updated_at 
		FROM tinyurl 
		WHERE short_code = $1`
	item := &feat.Tinyurl{}
	err := r.db.QueryRowContext(ctx, q, shortCode).Scan(
		&item.ID,
		&item.ShortCode,
		&item.OriginalURL,
		&item.ClickCount,
		&item.ExpiresAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, feat.ErrTinyurlNotFound
	}
	return item, err
}

func (r *pgRepository) IncrementClickCount(ctx context.Context, shortCode string) error {
	const q = `UPDATE tinyurl SET click_count = click_count + 1 WHERE short_code = $1`
	_, err := r.db.ExecContext(ctx, q, shortCode)
	return err
}

func (r *pgRepository) FindLatestShortCode(ctx context.Context) (string, error) {
	const q = `SELECT short_code FROM tinyurl ORDER BY created_at DESC LIMIT 1`
	var shortCode string
	err := r.db.QueryRowContext(ctx, q).Scan(&shortCode)
	if errors.Is(err, sql.ErrNoRows) {
		return "0", nil
	}
	return shortCode, err
}

func (r *pgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM tinyurl WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}
