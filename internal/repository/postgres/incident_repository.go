package postgres

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentRepository struct {
	pool *pgxpool.Pool
}

func NewIncidentRepository(pool *pgxpool.Pool) *IncidentRepository {
	return &IncidentRepository{
		pool: pool,
	}
}
func (r *IncidentRepository) GetOpenIncident(
	ctx context.Context,
	targetID int,
) (*domain.Incident, error) {
	query := `
		SELECT
			id,
			target_id,
			status,
			created_at
		FROM incidents
		WHERE target_id = $1
		  AND status = 'OPEN'
		LIMIT 1
	`

	var incident domain.Incident

	err := r.pool.QueryRow(
		ctx,
		query,
		targetID,
	).Scan(
		&incident.ID,
		&incident.TargetID,
		&incident.Status,
		&incident.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &incident, nil
}
func (r *IncidentRepository) CreateIncident(
	ctx context.Context,
	incident *domain.Incident,
) error {
	query := `
		INSERT INTO incidents (
			target_id,
			status,
			created_at
		)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		incident.TargetID,
		incident.Status,
		incident.CreatedAt,
	).Scan(&incident.ID)

	return err
}
func (r *IncidentRepository) CloseIncident(
	ctx context.Context,
	incidentID int,
	closedAt time.Time,
) error {
	query := `
		UPDATE incidents
		SET
			status = 'CLOSED',
			closed_at = $1
		WHERE id = $2
		  AND status = 'OPEN'
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		closedAt,
		incidentID,
	)

	return err
}

func (r *IncidentRepository) GetIncidentsForPeriod(
	ctx context.Context,
	targetID int,
	from time.Time,
	to time.Time,
) ([]domain.Incident, error) {
	query := `
        SELECT
            id,
            target_id,
            status,
            created_at,
            closed_at
        FROM incidents
        WHERE target_id = $1
          AND created_at < $3
          AND (
              closed_at IS NULL
              OR closed_at > $2
          )
        ORDER BY created_at
    `

	rows, err := r.pool.Query(
		ctx,
		query,
		targetID,
		from,
		to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []domain.Incident

	for rows.Next() {
		var incident domain.Incident
		var closedAt pgtype.Timestamptz

		err := rows.Scan(
			&incident.ID,
			&incident.TargetID,
			&incident.Status,
			&incident.CreatedAt,
			&closedAt,
		)
		if err != nil {
			return nil, err
		}

		if closedAt.Valid {
			incident.ClosedAt = closedAt.Time
		}

		incidents = append(incidents, incident)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return incidents, nil
}

func (r *IncidentRepository) ListByTargetID(
	ctx context.Context,
	targetID int,
	limit int,
) ([]domain.Incident, error) {
	query := `
		SELECT
			id,
			target_id,
			status,
			created_at,
			closed_at
		FROM incidents
		WHERE target_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(
		ctx,
		query,
		targetID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []domain.Incident

	for rows.Next() {
		var incident domain.Incident
		var closedAt pgtype.Timestamptz

		err := rows.Scan(
			&incident.ID,
			&incident.TargetID,
			&incident.Status,
			&incident.CreatedAt,
			&closedAt,
		)
		if err != nil {
			return nil, err
		}

		if closedAt.Valid {
			incident.ClosedAt = closedAt.Time
		}

		incidents = append(
			incidents,
			incident,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return incidents, nil
}
