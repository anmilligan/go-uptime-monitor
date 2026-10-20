package postgres

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertRepository struct {
	pool *pgxpool.Pool
}

func NewAlertRepository(pool *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{
		pool: pool,
	}
}

// CreateAlert inserts an alert if an alert with the same incident and type
// does not already exist. It returns true if a new alert was created.
func (r *AlertRepository) CreateAlert(ctx context.Context, alert *domain.Alert) (bool, error) {
	query := `
		INSERT INTO alerts (
			incident_id,
			type,
			message,
			created_at
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (incident_id, type) DO NOTHING
		RETURNING id
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		alert.IncidentID,
		alert.Type,
		alert.Message,
		alert.CreatedAt,
	).Scan(&alert.ID)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
