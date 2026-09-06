package postgres

import (
	"Uptime_Monitor/internal/domain"
	"context"

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

func (r *AlertRepository) CreateAlert(ctx context.Context, alert *domain.Alert) error {
	query := `
		INSERT INTO alerts (
			incident_id,
			type,
			message,
			created_at
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	return r.pool.QueryRow(
		ctx,
		query,
		alert.IncidentID,
		alert.Type,
		alert.Message,
		alert.CreatedAt,
	).Scan(&alert.ID)
}
