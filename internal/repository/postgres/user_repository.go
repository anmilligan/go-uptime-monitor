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

func (r *UserRepository) GetByID(
	ctx context.Context,
	id int,
) (*domain.User, error) {
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
