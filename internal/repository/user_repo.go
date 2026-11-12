package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/WenDoge/rest-api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserExists   = errors.New("user already exists")
	ErrEmailExists  = errors.New("email already exists")
	ErrUserNotFound = errors.New("user not found")
)

type ListUsersOptions struct {
	Filter     ListUsersFilter
	Navigation ListUsersNavigation
}

type ListUsersFilter struct {
	Username *string
}

type ListUsersNavigation struct {
	Limit  int
	Offset int
}

type UserRepository struct {
	pool *pgxpool.Pool
}

type UpdateUserParams struct {
	Username    *string
	Email       *string
	Description *string
	Age         *int
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (username, email, password, description, age)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, email, description, age, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query, user.Username, user.Email, user.Password, user.Description, user.Age).
		Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Description,
			&user.Age,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(err.Error(), "email") {
				return nil, ErrEmailExists
			}

			return nil, ErrUserExists
		}

		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, username, email, password, description, age, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	var user models.User

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Description,
		&user.Age,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, username, email, password, description, age, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var user models.User

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Description,
		&user.Age,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) List(ctx context.Context, opts ListUsersOptions) (*[]models.User, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filteredUserName := ""

	if opts.Filter.Username != nil {
		filteredUserName = *opts.Filter.Username
	}

	countQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE username ILIKE '%' || $1 || '%' AND deleted_at IS NULL
	`
	var total int

	if err := r.pool.QueryRow(ctx, countQuery, filteredUserName).Scan(&total); err != nil {
		return nil, total, err
	}

	query := `
		SELECT id, username, email, description, age, created_at, updated_at
		FROM users
		WHERE username ILIKE '%' || $1 || '%' AND deleted_at IS NULL
		ORDER BY created_at, id
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, filteredUserName, opts.Navigation.Limit, opts.Navigation.Offset)
	if err != nil {
		return nil, total, err
	}

	defer rows.Close()

	users := []models.User{}

	for rows.Next() {
		var user models.User

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Description,
			&user.Age,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, total, err
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, total, err
	}

	return &users, total, nil
}

func (r *UserRepository) Update(ctx context.Context, userID string, params UpdateUserParams) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user models.User

	query := `
	UPDATE users
	SET username = COALESCE($2, username), email = COALESCE($3, email), description = COALESCE($4, description), age = COALESCE($5,age), updated_at = NOW()
	WHERE id = $1 AND deleted_at IS NULL
	RETURNING id, username, email, description, age, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query, userID, params.Username, params.Email, params.Description, params.Age).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Description,
		&user.Age,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(err.Error(), "email") {
				return nil, ErrEmailExists
			}

			return nil, ErrUserExists
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user models.User

	query := `
		UPDATE users
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, username, email, description, age, created_at, updated_at, deleted_at
	`

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Description,
		&user.Age,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	query = `
		DELETE FROM refresh_tokens
		WHERE user_id = $1
	`

	if _, err = tx.Exec(ctx, query, id); err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &user, nil
}
