package token_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/shared/token"
)

const testSecret = "supersecretkey1234567890abcdefghij"
const testSalt = "$2a$10$examplepasswordhashfortesting"

func TestCreateAndVerifyToken_Access(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)
	userID := uuid.New()
	email := "test@example.com"

	tok, err := maker.CreateToken(userID, email, token.AccessToken, 15*time.Minute, testSalt)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := maker.VerifyToken(tok, testSalt)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, token.AccessToken, claims.Type)
}

func TestCreateAndVerifyToken_Refresh(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)
	userID := uuid.New()

	tok, err := maker.CreateToken(userID, "r@example.com", token.RefreshToken, 7*24*time.Hour, testSalt)
	require.NoError(t, err)

	claims, err := maker.VerifyToken(tok, testSalt)
	require.NoError(t, err)
	assert.Equal(t, token.RefreshToken, claims.Type)
}

func TestVerifyToken_Expired(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)
	tok, err := maker.CreateToken(uuid.New(), "e@example.com", token.AccessToken, -time.Minute, testSalt)
	require.NoError(t, err)

	_, err = maker.VerifyToken(tok, testSalt)
	assert.Error(t, err)
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	maker1 := token.NewJWTMaker(testSecret)
	maker2 := token.NewJWTMaker("differentsecret1234567890abcdefghij")

	tok, err := maker1.CreateToken(uuid.New(), "w@example.com", token.AccessToken, 15*time.Minute, testSalt)
	require.NoError(t, err)

	_, err = maker2.VerifyToken(tok, testSalt)
	assert.Error(t, err)
}

func TestVerifyToken_WrongSalt(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)

	tok, err := maker.CreateToken(uuid.New(), "s@example.com", token.AccessToken, 15*time.Minute, testSalt)
	require.NoError(t, err)

	_, err = maker.VerifyToken(tok, "differenthash")
	assert.Error(t, err)
}

func TestParseUnverifiedClaims_ReturnsUserID(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)
	userID := uuid.New()

	tok, err := maker.CreateToken(userID, "u@example.com", token.AccessToken, time.Minute, testSalt)
	require.NoError(t, err)

	claims, err := maker.ParseUnverifiedClaims(tok)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestParseUnverifiedClaims_MalformedToken(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)

	_, err := maker.ParseUnverifiedClaims("not.a.token")
	assert.Error(t, err)
}
