package posts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-boilerplate/app/shared/ports"
)

const (
	baseURL  = "https://jsonplaceholder.typicode.com"
	cacheTTL = 5 * time.Minute
)

var ErrPostNotFound = errors.New("post not found")

type CacheResult struct {
	Post   *Post
	Cached bool
}

type Service interface {
	GetPost(ctx context.Context, id int) (*CacheResult, error)
}

type service struct {
	http  ports.HTTPClient
	cache ports.Cache
}

func NewService(httpClient ports.HTTPClient, cache ports.Cache) Service {
	return &service{http: httpClient, cache: cache}
}

func (s *service) GetPost(ctx context.Context, id int) (*CacheResult, error) {
	key := fmt.Sprintf("posts:%d", id)

	if s.cache != nil {
		if val, err := s.cache.Get(ctx, key); err == nil {
			var post Post
			if err := json.Unmarshal([]byte(val), &post); err == nil {
				return &CacheResult{Post: &post, Cached: true}, nil
			}
		}
	}

	post, err := s.fetchFromUpstream(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if b, err := json.Marshal(post); err == nil {
			_ = s.cache.Set(ctx, key, string(b), cacheTTL)
		}
	}

	return &CacheResult{Post: post, Cached: false}, nil
}

func (s *service) fetchFromUpstream(ctx context.Context, id int) (*Post, error) {
	url := fmt.Sprintf("%s/posts/%d", baseURL, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrPostNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var post Post
	if err := json.Unmarshal(body, &post); err != nil {
		return nil, err
	}
	return &post, nil
}
