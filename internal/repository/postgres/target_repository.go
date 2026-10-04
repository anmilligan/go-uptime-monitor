package postgres

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TargetRepository struct {
	pool *pgxpool.Pool
}

func NewTargetRepository(pool *pgxpool.Pool) *TargetRepository {
	return &TargetRepository{
		pool: pool,
	}
}

func (r *TargetRepository) Save(ctx context.Context, target *domain.Target) error {
	query := `
    INSERT INTO targets (
        user_id,
        kind,
        address,
        interval_seconds,
        timeout_seconds,
        expected_status,
        expected_body
    )
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id, created_at, active
`
	err := r.pool.QueryRow(
		ctx,
		query,
		target.UserID,
		target.Kind,
		target.Address,
		int64(target.Interval.Seconds()),
		int64(target.Timeout.Seconds()),
		target.ExpectedStatus,
		target.ExpectedBody,
	).Scan(
		&target.ID,
		&target.CreatedAt,
		&target.Active,
	)
	if err != nil {
		return err
	}
	return nil
}
func (r *TargetRepository) GetByID(ctx context.Context, id int) (*domain.Target, error) {
	query := `
    SELECT
        id,
        user_id,
        kind,
        address,
        interval_seconds,
        timeout_seconds,
        expected_status,
        expected_body,
        created_at,
        active
    FROM targets
    WHERE id = $1
`
	target := &domain.Target{}

	var intervalSeconds int64
	var timeoutSeconds int64

	err := r.pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&target.ID,
		&target.UserID,
		&target.Kind,
		&target.Address,
		&intervalSeconds,
		&timeoutSeconds,
		&target.ExpectedStatus,
		&target.ExpectedBody,
		&target.CreatedAt,
		&target.Active,
	)

	if err != nil {
		return nil, err
	}

	target.Interval = time.Duration(intervalSeconds) * time.Second
	target.Timeout = time.Duration(timeoutSeconds) * time.Second

	return target, nil
}

func (r *TargetRepository) ListByUserID(ctx context.Context, userID int) ([]domain.Target, error) {
	query := `
	SELECT
		id,
		user_id,
		kind,
		address,
		interval_seconds,
		timeout_seconds,
		expected_status,
		expected_body,
		created_at,
		active
	FROM targets
	WHERE user_id = $1
	ORDER BY created_at DESC
`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []domain.Target

	for rows.Next() {
		var target domain.Target
		var intervalSeconds int64
		var timeoutSeconds int64

		err := rows.Scan(
			&target.ID,
			&target.UserID,
			&target.Kind,
			&target.Address,
			&intervalSeconds,
			&timeoutSeconds,
			&target.ExpectedStatus,
			&target.ExpectedBody,
			&target.CreatedAt,
			&target.Active,
		)
		if err != nil {
			return nil, err
		}

		target.Interval = time.Duration(intervalSeconds) * time.Second
		target.Timeout = time.Duration(timeoutSeconds) * time.Second

		targets = append(targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}

func (r *TargetRepository) Delete(ctx context.Context, id int) error {
	query := `
		DELETE FROM targets
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *TargetRepository) Update(
	ctx context.Context,
	target *domain.Target,
) error {
	query := `
		UPDATE targets
	SET
		kind = $1,
		address = $2,
		interval_seconds = $3,
		timeout_seconds = $4,
		expected_status = $5,
		expected_body = $6
	WHERE id = $7
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		target.Kind,
		target.Address,
		int64(target.Interval.Seconds()),
		int64(target.Timeout.Seconds()),
		target.ExpectedStatus,
		target.ExpectedBody,
		target.ID,
	)

	return err

}

func (r *TargetRepository) ListAll(
	ctx context.Context,
) ([]domain.Target, error) {
	query := `
        SELECT
            id,
            user_id,
            kind,
            address,
            interval_seconds,
            timeout_seconds,
            expected_status,
            expected_body,
            created_at,
            active
        FROM targets
        WHERE active = TRUE
        ORDER BY id
    `

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []domain.Target

	for rows.Next() {
		var target domain.Target
		var intervalSeconds int64
		var timeoutSeconds int64

		err := rows.Scan(
			&target.ID,
			&target.UserID,
			&target.Kind,
			&target.Address,
			&intervalSeconds,
			&timeoutSeconds,
			&target.ExpectedStatus,
			&target.ExpectedBody,
			&target.CreatedAt,
			&target.Active,
		)
		if err != nil {
			return nil, err
		}

		target.Interval = time.Duration(intervalSeconds) * time.Second
		target.Timeout = time.Duration(timeoutSeconds) * time.Second

		targets = append(targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}

func (r *TargetRepository) Pause(
	ctx context.Context,
	id int,
) error {
	query := `
		UPDATE targets
		SET active = FALSE
		WHERE id = $1
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		id,
	)

	return err
}

func (r *TargetRepository) Resume(
	ctx context.Context,
	id int,
) error {
	query := `
		UPDATE targets
		SET active = TRUE
		WHERE id = $1
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		id,
	)

	return err
}
