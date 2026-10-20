package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricsRepository struct {
	pool *pgxpool.Pool
}

func NewMetricsRepository(pool *pgxpool.Pool) *MetricsRepository {
	return &MetricsRepository{
		pool: pool,
	}
}

// GetMonitorStatusCounts returns the number of monitors grouped
// by their current derived status.
func (r *MetricsRepository) GetMonitorStatusCounts(ctx context.Context) (map[string]int, error) {
	const query = `
		WITH latest_checks AS (
			SELECT DISTINCT ON (target_id)
				target_id,
				is_success
			FROM checks
			ORDER BY target_id, checked_at DESC
		),
		monitor_statuses AS (
			SELECT
				CASE
					WHEN t.active = FALSE THEN 'paused'

					WHEN EXISTS (
						SELECT 1
						FROM incidents i
						WHERE i.target_id = t.id
						  AND i.status = 'OPEN'
					) THEN 'down'

					WHEN lc.target_id IS NULL THEN 'unknown'

					WHEN lc.is_success = TRUE THEN 'up'

					ELSE 'degraded'
				END AS status
			FROM targets t
			LEFT JOIN latest_checks lc
				ON lc.target_id = t.id
		)

		SELECT
			status,
			COUNT(*)
		FROM monitor_statuses
		GROUP BY status
	`

	rows, err := r.pool.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)

	for rows.Next() {
		var status string
		var count int64

		if err := rows.Scan(
			&status,
			&count,
		); err != nil {
			return nil, err
		}

		counts[status] = int(count)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return counts, nil
}

// GetOpenIncidentsCount returns the number of currently open incidents.
func (r *MetricsRepository) GetOpenIncidentsCount(ctx context.Context) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM incidents
		WHERE status = 'OPEN'
	`

	var count int

	if err := r.pool.QueryRow(
		ctx,
		query,
	).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// GetMonitorCreatedAt returns the creation time of each monitor
// keyed by its target ID.
func (r *MetricsRepository) GetMonitorCreatedAt(ctx context.Context) (map[int]time.Time, error) {
	const query = `
		SELECT id, created_at
		FROM targets
		ORDER BY id
	`

	rows, err := r.pool.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	monitors := make(map[int]time.Time)

	for rows.Next() {
		var id int
		var createdAt time.Time

		if err := rows.Scan(
			&id,
			&createdAt,
		); err != nil {
			return nil, err
		}

		monitors[id] = createdAt
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return monitors, nil
}
