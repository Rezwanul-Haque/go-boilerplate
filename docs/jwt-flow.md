# JWT Flow

Signing key = `JWT_SECRET + user.password_hash`  
Changing password changes the hash → all outstanding tokens become invalid instantly, no blocklist needed.

---

## Token Creation (Signup / Login)

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Service
    participant DB
    participant JWTMaker

    Client->>Handler: POST /signup or /login
    Handler->>Service: Signup(req) / Login(req)
    Service->>DB: FindByEmail → fetch user
    Service->>Service: bcrypt verify / hash password
    Service->>DB: Create user (signup only)
    Service->>JWTMaker: CreateToken(userID, email, AccessToken,  ttl, user.PasswordHash)
    JWTMaker-->>Service: signed access token
    Service->>JWTMaker: CreateToken(userID, email, RefreshToken, ttl, user.PasswordHash)
    JWTMaker-->>Service: signed refresh token
    Service-->>Handler: AuthResponse{access, refresh, user}
    Handler-->>Client: 200/201 JSON
```

**Signing key per call:** `HMAC-SHA256(JWT_SECRET + passwordHash)`

---

## Authenticated Request (change-password)

```mermaid
sequenceDiagram
    participant Client
    participant AuthMiddleware
    participant DB
    participant JWTMaker
    participant Handler

    Client->>AuthMiddleware: PUT /change-password  Bearer <token>
    AuthMiddleware->>JWTMaker: ParseUnverifiedClaims(token)
    Note over JWTMaker: Decode payload only — NO signature check yet
    JWTMaker-->>AuthMiddleware: claims{userID, ...}
    AuthMiddleware->>DB: FindByID(userID) → fetch user.PasswordHash
    AuthMiddleware->>JWTMaker: VerifyToken(token, user.PasswordHash)
    Note over JWTMaker: Recompute key = JWT_SECRET + hash, verify signature + expiry
    JWTMaker-->>AuthMiddleware: verified claims
    AuthMiddleware->>Handler: c.Set("claims", claims) → next()
    Handler-->>Client: 200 OK
```

---

## Refresh Token

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Service
    participant DB
    participant JWTMaker

    Client->>Handler: POST /refresh-token  {refresh_token}
    Handler->>Service: RefreshToken(req)
    Service->>JWTMaker: ParseUnverifiedClaims(refreshToken)
    Note over JWTMaker: Extract userID without verifying signature
    JWTMaker-->>Service: claims{userID}
    Service->>DB: FindByID(userID) → user.PasswordHash
    Service->>JWTMaker: VerifyToken(refreshToken, user.PasswordHash)
    JWTMaker-->>Service: verified claims
    Service->>Service: assert claims.Type == RefreshToken
    Service->>JWTMaker: CreateToken(..., AccessToken,  ttl, hash)
    Service->>JWTMaker: CreateToken(..., RefreshToken, ttl, hash)
    Service-->>Handler: new AuthResponse
    Handler-->>Client: 200 JSON
```

---

## Password Change → Token Invalidation

```mermaid
sequenceDiagram
    participant Client
    participant Service
    participant DB

    Note over Client,DB: Attacker holds a stolen token signed with old hash

    Client->>Service: ChangePassword(currentPass, newPass)
    Service->>DB: UpdatePassword → store NEW bcrypt hash
    Note over DB: user.password_hash changed

    Note over Client,DB: Attacker tries to use stolen token

    Client->>AuthMiddleware: Bearer <stolen_token>
    AuthMiddleware->>JWTMaker: ParseUnverifiedClaims → userID
    AuthMiddleware->>DB: FindByID → NEW hash
    AuthMiddleware->>JWTMaker: VerifyToken(token, NEW hash)
    JWTMaker-->>AuthMiddleware: ❌ invalid signature
    AuthMiddleware-->>Client: 401 Unauthorized
```

---

## Key Derivation Summary

| Step | Value |
|---|---|
| Base secret | `JWT_SECRET` (env var, never changes) |
| Per-user salt | `user.password_hash` (bcrypt, changes on password update) |
| Signing key | `JWT_SECRET + password_hash` (concatenated, used as HMAC-SHA256 key) |
| Token invalidated when | user changes password → hash changes → old signature invalid |

---

## Files

| File | Role |
|---|---|
| `app/shared/token/jwt.go` | `Maker` interface — `CreateToken`, `VerifyToken`, `ParseUnverifiedClaims` |
| `app/infra/middleware/auth.go` | Two-step verify: parse unverified → DB lookup → verify with hash |
| `app/features/users/service.go` | `buildAuthResponse` signs with hash; `RefreshToken` two-step verify |
| `cmd/main.go` | Wires `hashFn` closure over `usersRepo.FindByID` |
