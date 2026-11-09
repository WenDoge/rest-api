package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/WenDoge/rest-api/internal/config"
	"github.com/WenDoge/rest-api/internal/models"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"-"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"-"`
}

const (
	AccessTTL  = 15 * time.Minute
	RefreshTTL = 7 * 24 * time.Hour
)

func GenerateTokens(cfg *config.Config, user *models.User) (*TokenPair, error) {
	accessExpiresAt := time.Now().Add(AccessTTL)
	refreshExpiresAt := time.Now().Add(RefreshTTL)

	accessClaims := jwt.MapClaims{
		"jti":     uuid.NewString(),
		"type":    "access",
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     accessExpiresAt.Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)

	accessTokenString, err := accessToken.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("Failed to generate accessToken")
	}

	refreshClaims := jwt.MapClaims{
		"jti":     uuid.NewString(),
		"type":    "refresh",
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     refreshExpiresAt.Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	refreshTokenString, err := refreshToken.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("Failed to generate refreshToken")
	}

	return &TokenPair{
		AccessToken:      accessTokenString,
		RefreshToken:     refreshTokenString,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func ParseJWT(cfg *config.Config, tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
