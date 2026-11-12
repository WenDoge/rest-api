package handlers

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/WenDoge/rest-api/internal/config"
	"github.com/WenDoge/rest-api/internal/models"
	"github.com/WenDoge/rest-api/internal/repository"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	user := &models.User{ID: "1", Email: "user@example.com", Password: string(hash)}

	tests := []struct {
		name       string
		body       string
		users      *mockUserRepo
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{bad json`,
			users:      &mockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "user not found",
			body:       `{"email":"user@example.com","password":"secret"}`,
			users:      &mockUserRepo{err: repository.ErrUserNotFound},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong password",
			body:       `{"email":"user@example.com","password":"wrong"}`,
			users:      &mockUserRepo{user: user},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "success",
			body:       `{"email":"user@example.com","password":"secret"}`,
			users:      &mockUserRepo{user: user},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAuthHandler(tt.users, &mockTokenRepo{}, cfg)
			router := gin.New()
			router.POST("/auth/login", h.Login)

			w := doRequest(router, http.MethodPost, "/auth/login", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK && !strings.Contains(w.Body.String(), "access_token") {
				t.Errorf("body has no access_token: %s", w.Body.String())
			}
		})
	}
}

func TestRegister(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}

	user := &models.User{ID: "1", Email: "user@example.com"}

	tests := []struct {
		name       string
		body       string
		users      *mockUserRepo
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{bad json`,
			users:      &mockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "success",
			body:       `{"email":"user@example.com", "username": "user","password":"secret"}`,
			users:      &mockUserRepo{user: user},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "user already exists",
			body:       `{"email":"user@example.com", "username": "user","password":"secret"}`,
			users:      &mockUserRepo{err: repository.ErrUserExists},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "email already exists",
			body:       `{"email":"user@example.com", "username": "user","password":"secret"}`,
			users:      &mockUserRepo{err: repository.ErrEmailExists},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "unhandled error",
			body:       `{"email":"user@example.com", "username": "user","password":"secret"}`,
			users:      &mockUserRepo{err: errors.New("unknown error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAuthHandler(tt.users, &mockTokenRepo{}, cfg)
			router := gin.New()
			router.POST("/auth/register", h.Register)

			w := doRequest(router, http.MethodPost, "/auth/register", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusCreated && !strings.Contains(w.Body.String(), "access_token") {
				t.Errorf("body has no access_token: %s", w.Body.String())
			}
		})
	}
}
