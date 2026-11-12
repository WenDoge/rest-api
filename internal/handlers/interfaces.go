package handlers

import (
	"context"

	"github.com/WenDoge/rest-api/internal/models"
	"github.com/WenDoge/rest-api/internal/repository"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	List(ctx context.Context, opts repository.ListUsersOptions) (*[]models.User, int, error)
	Create(ctx context.Context, user *models.User) (*models.User, error)
	Delete(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, userID string, params repository.UpdateUserParams) (*models.User, error)
}

type TokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	Rotate(ctx context.Context, oldTokenHash string, token *models.RefreshToken) error
	Delete(ctx context.Context, tokenHash string) error
}
