package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/WenDoge/rest-api/internal/config"
	"github.com/WenDoge/rest-api/internal/models"
	"github.com/golang-jwt/jwt/v4"
)

func TestGenerateTokens(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	user := &models.User{ID: "userID", Email: "email@email.com"}

	tokens, err := GenerateTokens(cfg, user)
	if err != nil {
		t.Fatalf("GenerateTokens: %v", err)
	}

	accessToken, err := ParseJWT(cfg, tokens.AccessToken)

	if err != nil || !accessToken.Valid {
		t.Fatalf("ParseJWT(access): %v", err)
	}

	accessClaims := accessToken.Claims.(jwt.MapClaims)

	refreshToken, err := ParseJWT(cfg, tokens.RefreshToken)

	if err != nil || !refreshToken.Valid {
		t.Fatalf("ParseJWT(refresh): %v", err)
	}

	refreshClaims := refreshToken.Claims.(jwt.MapClaims)

	t.Run("access", func(t *testing.T) {
		t.Run("type is access", func(t *testing.T) {
			if accessClaims["type"] != "access" {
				t.Errorf("type = %v, want access", accessClaims["type"])
			}
		})

		t.Run("user_id from user", func(t *testing.T) {
			if accessClaims["user_id"] != "userID" {
				t.Errorf("user_id = %v, want userID", accessClaims["user_id"])
			}
		})

		t.Run("email from user", func(t *testing.T) {
			if accessClaims["email"] != "email@email.com" {
				t.Errorf("email = %v, want email@email.com", accessClaims["email"])
			}
		})
	})

	t.Run("refresh", func(t *testing.T) {
		t.Run("type is refresh", func(t *testing.T) {
			if refreshClaims["type"] != "refresh" {
				t.Errorf("type = %v, want refresh", refreshClaims["type"])
			}
		})

		t.Run("user_id from user", func(t *testing.T) {
			if refreshClaims["user_id"] != "userID" {
				t.Errorf("user_id = %v, want userID", refreshClaims["user_id"])
			}
		})

		t.Run("email from user", func(t *testing.T) {
			if refreshClaims["email"] != "email@email.com" {
				t.Errorf("email = %v, want email@email.com", refreshClaims["email"])
			}
		})
	})
}

func TestParseJWT(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	otherCfg := &config.Config{JWTSecret: "other-secret"}

	unexpired := time.Now().Add(time.Hour).Unix()
	expired := time.Now().Add(-time.Minute).Unix()

	tokens, err := GenerateTokens(cfg, &models.User{ID: "user-A", Email: "mail@mail.com"})
	if err != nil {
		t.Fatalf("GenerateTokens: %v", err)
	}

	otherTokens, err := GenerateTokens(otherCfg, &models.User{ID: "user-A", Email: "mail@mail.com"})
	if err != nil {
		t.Fatalf("GenerateTokens(otherCfg): %v", err)
	}

	noSign, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"exp": unexpired}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("no sign: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		wantErrIs error
	}{
		{name: "valid access", token: tokens.AccessToken},
		{name: "valid refresh", token: tokens.RefreshToken},

		{name: "wrong secret", token: otherTokens.AccessToken, wantErr: true},

		{
			name:    "HS512 instead of HS256",
			token:   signToken(t, jwt.SigningMethodHS512, cfg.JWTSecret, jwt.MapClaims{"exp": unexpired}),
			wantErr: true,
		},
		{name: "no sign", token: noSign, wantErr: true},

		{
			name:      "expired",
			token:     signToken(t, jwt.SigningMethodHS256, cfg.JWTSecret, jwt.MapClaims{"exp": expired}),
			wantErr:   true,
			wantErrIs: jwt.ErrTokenExpired,
		},
		{
			name: "not valid yet (nbf in future)",
			token: signToken(t, jwt.SigningMethodHS256, cfg.JWTSecret, jwt.MapClaims{
				"exp": unexpired,
				"nbf": unexpired,
			}),
			wantErr:   true,
			wantErrIs: jwt.ErrTokenNotValidYet,
		},

		{name: "garbage", token: "notJWT", wantErr: true},
		{name: "empty", token: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ParseJWT(cfg, tt.token)

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("err = %v, want errors.Is(err, %v)", err, tt.wantErrIs)
			}

			if !tt.wantErr && !token.Valid {
				t.Errorf("token.Valid = false, want true")
			}
		})
	}
}

func signToken(t *testing.T, method jwt.SigningMethod, secret string, claims jwt.MapClaims) string {
	t.Helper()

	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signToken: %v", err)
	}

	return token
}
