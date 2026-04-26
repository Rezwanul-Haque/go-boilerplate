package tinyurl

import (
	"context"
	"errors"
	"fmt"
	"go-boilerplate/app/shared/ports"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"go-boilerplate/app/shared/model"
)

const (
	seqKey      = "tinyurl:seq"
	cacheKeyFmt = "tinyurl:%s"
	alphabet    = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	maxRetries  = 3
	defaultTTL  = 30 * 24 * time.Hour
)

type Service interface {
	Create(ctx context.Context, req CreateTinyurlRequest) (*TinyurlResponse, error)
	List(ctx context.Context, limit, offset int) ([]*TinyurlResponse, error)
	Redirect(ctx context.Context, shortCode string) (string, error)
}

type service struct {
	repo  Repository
	cache ports.Cache
}

func NewService(repo Repository, cache ports.Cache) Service {
	return &service{
		repo:  repo,
		cache: cache,
	}
}

func (s *service) Create(ctx context.Context, req CreateTinyurlRequest) (*TinyurlResponse, error) {
	for i := 0; i < maxRetries; i++ {
		seqVal, err := s.cache.Incr(ctx, seqKey)
		if err != nil {
			return nil, err
		}

		item := &Tinyurl{
			Base:        model.Base{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			ShortCode:   base62Encode(seqVal),
			OriginalURL: req.OriginalURL,
			ExpiresAt:   time.Now().Add(defaultTTL),
		}

		if err := s.repo.Create(ctx, item); err != nil {
			if isUniqueViolation(err) {
				if syncErr := s.resyncSeq(ctx); syncErr != nil {
					return nil, syncErr
				}
				continue
			}
			return nil, err
		}
		return toResponse(item), nil
	}
	return nil, errors.New("failed to generate unique short code after retries")
}

func (s *service) List(ctx context.Context, limit, offset int) ([]*TinyurlResponse, error) {
	items, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]*TinyurlResponse, len(items))
	for i, item := range items {
		result[i] = toResponse(item)
	}
	return result, nil
}

func (s *service) Redirect(ctx context.Context, shortCode string) (string, error) {
	key := fmt.Sprintf(cacheKeyFmt, shortCode)

	originalURL, err := s.cache.Get(ctx, key)
	if err == nil {
		go func() {
			_ = s.repo.IncrementClickCount(context.Background(), shortCode)
		}()
		return originalURL, nil
	}
	if !errors.Is(err, ports.ErrCacheMiss) {
		return "", err
	}

	item, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}
	if time.Now().After(item.ExpiresAt) {
		return "", ErrTinyurlExpired
	}

	ttl := time.Until(item.ExpiresAt)
	_ = s.cache.Set(ctx, key, item.OriginalURL, ttl)
	_ = s.repo.IncrementClickCount(ctx, shortCode)

	return item.OriginalURL, nil
}

func (s *service) resyncSeq(ctx context.Context) error {
	shortCode, err := s.repo.FindLatestShortCode(ctx)
	if err != nil {
		return err
	}
	maxSeq := base62Decode(shortCode)
	return s.cache.Set(ctx, seqKey, maxSeq, 0)
}

func toResponse(item *Tinyurl) *TinyurlResponse {
	return &TinyurlResponse{
		ID:          item.ID.String(),
		ShortCode:   item.ShortCode,
		OriginalURL: item.OriginalURL,
		ClickCount:  item.ClickCount,
		ExpiresAt:   item.ExpiresAt,
		CreatedAt:   item.CreatedAt,
	}
}

// secret > 62^5 (916132832) so XOR gives 6-char codes for all small seq values
const secret = int64(0x5f5f5f5f)

func base62Encode(n int64) string {
	val := n ^ secret
	var result []byte
	for val > 0 {
		result = append([]byte{alphabet[val%62]}, result...)
		val /= 62
	}
	return string(result)
}

func base62Decode(s string) int64 {
	var val int64
	for _, c := range s {
		val = val*62 + int64(strings.IndexRune(alphabet, c))
	}
	return val ^ secret
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
