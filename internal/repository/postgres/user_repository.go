package postgres

import (
	"Uptime_Monitor/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

// GetByID returns a user by its internal ID.
func (r *UserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	query := `
		SELECT
			id,
			telegram_id,
			username,
			created_at
		FROM users
		WHERE id = $1
	`

	var user domain.User

	err := r.pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Upsert creates a user or updates the username of an existing user
// identified by the same Telegram ID.
func (r *UserRepository) Upsert(ctx context.Context, user *domain.User) error {
	query := `
        INSERT INTO users (
            telegram_id,
            username
        )
        VALUES ($1, $2)
        ON CONFLICT (telegram_id)
        DO UPDATE SET
            username = EXCLUDED.username
        RETURNING id, created_at
    `

	return r.pool.QueryRow(
		ctx,
		query,
		user.TelegramID,
		user.Username,
	).Scan(
		&user.ID,
		&user.CreatedAt,
	)
}
