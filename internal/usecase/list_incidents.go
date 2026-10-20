package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
)

type IListIncidentsRepository interface {
	ListByTargetID(
		ctx context.Context,
		targetID int,
		limit int,
	) ([]domain.Incident, error)
}

type ListIncidentsUseCase struct {
	incidentRepo IListIncidentsRepository
}

func NewListIncidentsUseCase(incidentRepo IListIncidentsRepository) *ListIncidentsUseCase {
	return &ListIncidentsUseCase{
		incidentRepo: incidentRepo,
	}
}

// ListIncidents returns the most recent incidents for the specified target.
func (uc *ListIncidentsUseCase) ListIncidents(ctx context.Context, targetID int, limit int) ([]domain.Incident, error) {
	if targetID <= 0 {
		return nil, errors.New(
			"target ID must be greater than zero",
		)
	}

	if limit <= 0 {
		return nil, errors.New(
			"limit must be greater than zero",
		)
	}

	return uc.incidentRepo.ListByTargetID(
		ctx,
		targetID,
		limit,
	)
}
