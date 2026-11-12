package repository

import (
	"context"
	"errors"
	"time"

	"github.com/WenDoge/rest-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTokenNotFound = errors.New("token not found")

type TokenRepository struct {
	pool *pgxpool.Pool
}

func NewTokenRepository(pool *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{pool: pool}
}

func (r *TokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`

	_, err := r.pool.Exec(ctx, query, token.UserID, token.TokenHash, token.ExpiresAt)

	return err
}

func (r *TokenRepository) Rotate(
	ctx context.Context,
	oldTokenHash string,
	token *models.RefreshToken,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	query := `
		DELETE FROM refresh_tokens
		WHERE user_id = $1 AND token_hash = $2 AND expires_at > NOW()
	`

	commandTag, err := tx.Exec(ctx, query, token.UserID, oldTokenHash)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}

	query = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)

	`

	_, err = tx.Exec(ctx, query, token.UserID, token.TokenHash, token.ExpiresAt)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *TokenRepository) Delete(ctx context.Context, tokenHash string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		DELETE FROM refresh_tokens
		WHERE token_hash = $1
	`
	commandTag, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}

	return nil
}
