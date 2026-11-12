package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WenDoge/rest-api/internal/models"
	"github.com/WenDoge/rest-api/internal/repository"
	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode) // убирает логи gin из вывода тестов
	os.Exit(m.Run())
}

type mockUserRepo struct {
	user  *models.User
	users []models.User
	err   error
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) List(ctx context.Context, opts repository.ListUsersOptions) (*[]models.User, int, error) {
	return &m.users, len(m.users), m.err
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) (*models.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) Delete(ctx context.Context, id string) (*models.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) Update(
	ctx context.Context,
	userID string,
	params repository.UpdateUserParams,
) (*models.User, error) {
	return m.user, m.err
}

type mockTokenRepo struct {
	err error
}

func (m *mockTokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	return m.err
}

func (m *mockTokenRepo) Rotate(ctx context.Context, oldTokenHash string, token *models.RefreshToken) error {
	return m.err
}

func (m *mockTokenRepo) Delete(ctx context.Context, tokenHash string) error {
	return m.err
}

func doRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
