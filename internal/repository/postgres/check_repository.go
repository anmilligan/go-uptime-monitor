package postgres

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckRepository struct {
	pool *pgxpool.Pool
}

func NewCheckRepository(pool *pgxpool.Pool) *CheckRepository {
	return &CheckRepository{
		pool: pool,
	}
}

func (r *CheckRepository) SaveCheck(
	ctx context.Context,
	check *domain.Check,
) error {
	query := `
        INSERT INTO checks (
            target_id,
            status,
            answer,
            response_time_ms,
            is_success,
            checked_at
        )
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `

	err := r.pool.QueryRow(
		ctx,
		query,
		check.TargetID,
		check.Status,
		check.Answer,
		check.ResponseTime.Milliseconds(),
		check.IsSuccess,
		check.CheckedAt,
	).Scan(&check.ID)

	return err
}
func (r *CheckRepository) GetLastChecks(
	ctx context.Context,
	targetID int,
	limit int,
) ([]domain.Check, error) {
	query := `
        SELECT
            id,
            target_id,
            status,
            answer,
            response_time_ms,
            is_success,
            checked_at
        FROM checks
        WHERE target_id = $1
        ORDER BY checked_at DESC
        LIMIT $2
    `

	rows, err := r.pool.Query(ctx, query, targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []domain.Check

	for rows.Next() {
		var check domain.Check
		var responseTimeMs int64

		if err := rows.Scan(
			&check.ID,
			&check.TargetID,
			&check.Status,
			&check.Answer,
			&responseTimeMs,
			&check.IsSuccess,
			&check.CheckedAt,
		); err != nil {
			return nil, err
		}

		check.ResponseTime =
			time.Duration(responseTimeMs) * time.Millisecond

		checks = append(checks, check)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return checks, nil
}
