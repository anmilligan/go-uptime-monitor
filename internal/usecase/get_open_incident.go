package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
)

type IOpenIncidentRepository interface {
	GetOpenIncident(
		ctx context.Context,
		targetID int,
	) (*domain.Incident, error)
}

type GetOpenIncidentUseCase struct {
	incidentRepo IOpenIncidentRepository
}

func NewGetOpenIncidentUseCase(
	incidentRepo IOpenIncidentRepository,
) *GetOpenIncidentUseCase {
	return &GetOpenIncidentUseCase{
		incidentRepo: incidentRepo,
	}
}

func (uc *GetOpenIncidentUseCase) GetOpenIncident(
	ctx context.Context,
	targetID int,
) (*domain.Incident, error) {
	return uc.incidentRepo.GetOpenIncident(ctx, targetID)
}
