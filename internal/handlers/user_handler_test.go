package handlers

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/WenDoge/rest-api/internal/config"
	"github.com/WenDoge/rest-api/internal/middleware"
	"github.com/WenDoge/rest-api/internal/models"
	"github.com/WenDoge/rest-api/internal/repository"
	"github.com/gin-gonic/gin"
)

func withUserID(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID != "" {
			c.Set(middleware.UserIDKey, userID)
		}
		c.Next()
	}
}

func TestGetMe(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	user := &models.User{ID: "1", Email: "user@example.com", Username: "user"}

	tests := []struct {
		name       string
		userID     string
		users      *mockUserRepo
		wantStatus int
	}{
		{
			name:       "success",
			userID:     "1",
			users:      &mockUserRepo{user: user},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no user id in context",
			userID:     "",
			users:      &mockUserRepo{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user not found",
			userID:     "1",
			users:      &mockUserRepo{err: repository.ErrUserNotFound},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unhandled error",
			userID:     "1",
			users:      &mockUserRepo{err: errors.New("unknown error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUserHandler(tt.users, cfg)
			router := gin.New()
			router.Use(withUserID(tt.userID))
			router.GET("/users/my", h.GetMe)

			w := doRequest(router, http.MethodGet, "/users/my", "")

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK && !strings.Contains(w.Body.String(), user.Email) {
				t.Errorf("body has no user email: %s", w.Body.String())
			}
		})
	}
}

func TestList(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	users := []models.User{
		{ID: "1", Email: "first@example.com", Username: "first"},
		{ID: "2", Email: "second@example.com", Username: "second"},
	}

	tests := []struct {
		name       string
		path       string
		users      *mockUserRepo
		wantStatus int
	}{
		{
			name:       "success",
			path:       "/users",
			users:      &mockUserRepo{users: users},
			wantStatus: http.StatusOK,
		},
		{
			name:       "success with filter and pagination",
			path:       "/users?username=first&limit=10&offset=0",
			users:      &mockUserRepo{users: users[:1]},
			wantStatus: http.StatusOK,
		},
		{
			name:       "limit too big",
			path:       "/users?limit=1000",
			users:      &mockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit is not a number",
			path:       "/users?limit=abc",
			users:      &mockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unhandled error",
			path:       "/users",
			users:      &mockUserRepo{err: errors.New("unknown error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUserHandler(tt.users, cfg)
			router := gin.New()
			router.Use(withUserID("1"))
			router.GET("/users", h.List)

			w := doRequest(router, http.MethodGet, tt.path, "")

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK && !strings.Contains(w.Body.String(), `"items"`) {
				t.Errorf("body has no items: %s", w.Body.String())
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	user := &models.User{ID: "1", Email: "user@example.com", Username: "newname"}

	tests := []struct {
		name       string
		userID     string
		body       string
		users      *mockUserRepo
		wantStatus int
	}{
		{
			name:       "success",
			userID:     "1",
			body:       `{"username":"newname"}`,
			users:      &mockUserRepo{user: user},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			userID:     "1",
			body:       `{bad json`,
			users:      &mockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid email",
			userID:     "1",
			body:       `{"email":"not-an-email"}`,
			users:      &mockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no user id in context",
			userID:     "",
			body:       `{"username":"newname"}`,
			users:      &mockUserRepo{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user not found",
			userID:     "1",
			body:       `{"username":"newname"}`,
			users:      &mockUserRepo{err: repository.ErrUserNotFound},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "email already exists",
			userID:     "1",
			body:       `{"email":"taken@example.com"}`,
			users:      &mockUserRepo{err: repository.ErrEmailExists},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "username already exists",
			userID:     "1",
			body:       `{"username":"taken"}`,
			users:      &mockUserRepo{err: repository.ErrUserExists},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "unhandled error",
			userID:     "1",
			body:       `{"username":"newname"}`,
			users:      &mockUserRepo{err: errors.New("unknown error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUserHandler(tt.users, cfg)
			router := gin.New()
			router.Use(withUserID(tt.userID))
			router.PATCH("/users/my", h.Update)

			w := doRequest(router, http.MethodPatch, "/users/my", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK && !strings.Contains(w.Body.String(), user.Username) {
				t.Errorf("body has no updated username: %s", w.Body.String())
			}
		})
	}
}

func TestDelete(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	user := &models.User{ID: "1", Email: "user@example.com", Username: "user"}

	tests := []struct {
		name       string
		userID     string
		users      *mockUserRepo
		wantStatus int
	}{
		{
			name:       "success",
			userID:     "1",
			users:      &mockUserRepo{user: user},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no user id in context",
			userID:     "",
			users:      &mockUserRepo{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user not found",
			userID:     "1",
			users:      &mockUserRepo{err: repository.ErrUserNotFound},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unhandled error",
			userID:     "1",
			users:      &mockUserRepo{err: errors.New("unknown error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUserHandler(tt.users, cfg)
			router := gin.New()
			router.Use(withUserID(tt.userID))
			router.DELETE("/users/my", h.Delete)

			w := doRequest(router, http.MethodDelete, "/users/my", "")

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK &&
				!strings.Contains(w.Header().Get("Set-Cookie"), refreshCookieName+"=;") {
				t.Errorf("refresh cookie was not cleared; Set-Cookie: %q", w.Header().Get("Set-Cookie"))
			}
		})
	}
}
