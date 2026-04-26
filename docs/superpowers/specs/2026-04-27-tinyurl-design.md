# TinyURL Feature — Design Spec

## Overview

URL shortener feature. Public access (no auth). Generates collision-free short codes via Redis counter + base62
encoding. Tracks click counts. Supports 302 redirects at root level.

---

## Data Model

```go
type Tinyurl struct {
    model.Base                           // id (UUID), created_at, updated_at
    ShortCode   string    `db:"short_code"`
    OriginalURL string    `db:"original_url"`
    ClickCount  int64     `db:"click_count"`
    ExpiresAt   time.Time `db:"expires_at"`
}
```

### Migration (000003_create_tinyurl.up.sql)

```sql
CREATE TABLE IF NOT EXISTS tinyurl (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    short_code   VARCHAR(10) UNIQUE NOT NULL,
    original_url TEXT        NOT NULL,
    click_count  BIGINT      NOT NULL DEFAULT 0,
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tinyurl_short_code ON tinyurl (short_code);
CREATE INDEX idx_tinyurl_expires_at  ON tinyurl (expires_at);
```

No PostgreSQL sequence — counter lives in Redis.

---

## API Endpoints

| Method   | Path                       | Description                          |
|----------|----------------------------|--------------------------------------|
| `POST`   | `/api/v1/tinyurl`          | Create short URL                     |
| `GET`    | `/api/v1/tinyurl`          | List all URLs (paginated)            |
| `GET`    | `/:short_code`             | 302 redirect + increment click count |

Redirect endpoint is registered on the root Echo group, not under `/api/v1`.

---

## Short Code Generation

**Algorithm:** Redis counter → base62 encode

```
Redis key:  "tinyurl:seq"
Alphabet:   "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
```

**On create:**
1. `INCR tinyurl:seq` → get `seqVal`
2. base62 encode `seqVal` → `short_code`
3. `INSERT INTO tinyurl ...`

**Startup initialization:**
```
SET tinyurl:seq 0 NX
```
Only sets if key missing — preserves existing counter.

**Collision recovery (Redis flush scenario):**
1. INSERT fails with unique violation on `short_code`
2. Query DB: `SELECT short_code FROM tinyurl ORDER BY created_at DESC LIMIT 1`
3. Decode base62 → `maxSeq`
4. `SET tinyurl:seq <maxSeq>` (overwrite stale counter)
5. `INCR` → retry INSERT (max 3 attempts)

---

## Redis Cache Flow

**Key:** `tinyurl:<short_code>`  
**Value:** `original_url` (string)  
**TTL:** matches `expires_at` of the record

### Redirect flow:
```
GET /:short_code
  → GET tinyurl:<short_code> from Redis
  → HIT:  302 redirect + fire-and-forget goroutine: UPDATE click_count+1 in DB
  → MISS: SELECT from DB WHERE short_code=? AND expires_at > NOW()
      → NOT FOUND / EXPIRED: 404
      → FOUND: SET Redis key with remaining TTL → 302 redirect + increment click_count
```

### Invalidation:
- `PUT /:id` → delete Redis key, update DB
- `DELETE /:id` → delete Redis key, delete from DB

---

## DTOs

```go
type CreateTinyurlRequest struct {
    OriginalURL string `json:"original_url" validate:"required,url"`
}

type TinyurlResponse struct {
    ID          string    `json:"id"`
    ShortCode   string    `json:"short_code"`
    OriginalURL string    `json:"original_url"`
    ClickCount  int64     `json:"click_count"`
    ExpiresAt   time.Time `json:"expires_at"`
    CreatedAt   time.Time `json:"created_at"`
}
```

---

## Error Handling

| Scenario                  | Response          |
|---------------------------|-------------------|
| Short code not found      | 404 Not Found     |
| URL expired               | 404 Not Found     |
| Invalid URL format        | 400 Bad Request   |
| Duplicate short code (DB) | retry (internal)  |

---

## Layers

```
Handler → Service → Repository → PostgreSQL
              ↓
           Redis (seq counter + URL cache)
```

- **Handler:** bind/validate request, call service, return response
- **Service:** short code generation, cache read/write, click count increment
- **Repository:** `Create`, `List`, `FindByShortCode`, `IncrementClickCount`
- **Routes:** Create + List under `/api/v1/tinyurl`, redirect at root `/:short_code`
