package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

type Maker interface {
	CreateToken(userID uuid.UUID, email string, tokenType TokenType, ttl time.Duration, salt string) (string, error)
	VerifyToken(tokenStr string, salt string) (*Claims, error)
	ParseUnverifiedClaims(tokenStr string) (*Claims, error)
}

type jwtMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) Maker {
	return &jwtMaker{secretKey: secretKey}
}

func (m *jwtMaker) signingKey(salt string) []byte {
	return []byte(m.secretKey + salt)
}

func (m *jwtMaker) CreateToken(userID uuid.UUID, email string, tokenType TokenType, ttl time.Duration, salt string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.signingKey(salt))
}

func (m *jwtMaker) VerifyToken(tokenStr string, salt string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.signingKey(salt), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *jwtMaker) ParseUnverifiedClaims(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenStr, claims)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
