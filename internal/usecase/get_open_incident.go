package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
)

type IOpenIncidentRepository interface {
	GetOpenIncident(ctx context.Context, targetID int) (*domain.Incident, error)
}
type GetOpenIncidentUseCase struct {
	incidentRepo IOpenIncidentRepository
}

func NewGetOpenIncidentUseCase(incidentRepo IOpenIncidentRepository) *GetOpenIncidentUseCase {
	return &GetOpenIncidentUseCase{
		incidentRepo: incidentRepo,
	}
}

// GetOpenIncident returns the currently open incident for the specified target.
// It returns nil without an error if there is no open incident.
func (uc *GetOpenIncidentUseCase) GetOpenIncident(ctx context.Context, targetID int) (*domain.Incident, error) {
	if targetID <= 0 {
		return nil, errors.New("target ID must be greater than zero")
	}

	return uc.incidentRepo.GetOpenIncident(ctx, targetID)
}
