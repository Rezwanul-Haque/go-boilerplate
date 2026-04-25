package posts_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/features/posts"
	"go-boilerplate/app/shared/ports"
)

// --- mock HTTPClient ---

type mockHTTP struct {
	resp *http.Response
	err  error
}

func (m *mockHTTP) Do(_ *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

func jsonResp(code int, body any) *http.Response {
	b, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(string(b))),
	}
}

// --- mock Cache ---

type mockCache struct {
	store map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]string)}
}

func (m *mockCache) Get(_ context.Context, key string) (string, error) {
	v, ok := m.store[key]
	if !ok {
		return "", ports.ErrCacheMiss
	}
	return v, nil
}

func (m *mockCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
	m.store[key] = value.(string)
	return nil
}

func (m *mockCache) Delete(_ context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func (m *mockCache) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.store[key]
	return ok, nil
}

func (m *mockCache) Ping(_ context.Context) error { return nil }

// --- tests ---

func TestGetPost_CacheMiss_FetchesUpstream(t *testing.T) {
	upstream := &posts.Post{ID: 1, UserID: 1, Title: "hello", Body: "world"}
	h := &mockHTTP{resp: jsonResp(http.StatusOK, upstream)}
	c := newMockCache()
	svc := posts.NewService(h, c)

	result, err := svc.GetPost(context.Background(), 1)

	require.NoError(t, err)
	assert.False(t, result.Cached)
	assert.Equal(t, 1, result.Post.ID)
	assert.Equal(t, "hello", result.Post.Title)
}

func TestGetPost_CacheHit_ReturnsFromCache(t *testing.T) {
	upstream := &posts.Post{ID: 2, UserID: 1, Title: "cached", Body: "body"}
	b, _ := json.Marshal(upstream)
	c := newMockCache()
	c.store["posts:2"] = string(b)

	h := &mockHTTP{} // must not be called
	svc := posts.NewService(h, c)

	result, err := svc.GetPost(context.Background(), 2)

	require.NoError(t, err)
	assert.True(t, result.Cached)
	assert.Equal(t, "cached", result.Post.Title)
}

func TestGetPost_UpstreamNotFound_ReturnsErrPostNotFound(t *testing.T) {
	h := &mockHTTP{resp: jsonResp(http.StatusNotFound, nil)}
	svc := posts.NewService(h, newMockCache())

	_, err := svc.GetPost(context.Background(), 999)

	assert.ErrorIs(t, err, posts.ErrPostNotFound)
}

func TestGetPost_UpstreamError_ReturnsError(t *testing.T) {
	h := &mockHTTP{err: errors.New("connection refused")}
	svc := posts.NewService(h, newMockCache())

	_, err := svc.GetPost(context.Background(), 1)

	assert.Error(t, err)
}

func TestGetPost_NilCache_FetchesUpstreamEachTime(t *testing.T) {
	upstream := &posts.Post{ID: 3, Title: "no cache"}
	h := &mockHTTP{resp: jsonResp(http.StatusOK, upstream)}
	svc := posts.NewService(h, nil)

	result, err := svc.GetPost(context.Background(), 3)

	require.NoError(t, err)
	assert.False(t, result.Cached)
}

func TestGetPost_PopulatesCache_AfterFetch(t *testing.T) {
	upstream := &posts.Post{ID: 4, Title: "populate"}
	h := &mockHTTP{resp: jsonResp(http.StatusOK, upstream)}
	c := newMockCache()
	svc := posts.NewService(h, c)

	_, err := svc.GetPost(context.Background(), 4)
	require.NoError(t, err)

	_, cached := c.store["posts:4"]
	assert.True(t, cached)
}
