package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"
)

type IAlertRepository interface {
	CreateAlert(ctx context.Context, alert *domain.Alert) error
}

type CreateAlertUseCase struct {
	repo IAlertRepository
}

func NewCreateAlertUseCase(repo IAlertRepository) *CreateAlertUseCase {
	return &CreateAlertUseCase{
		repo: repo,
	}
}

func (uc *CreateAlertUseCase) CreateAlert(ctx context.Context, incidentID int, alertType string, message string) error {
	alert := domain.Alert{
		IncidentID: incidentID,
		Type:       alertType,
		Message:    message,
		CreatedAt:  time.Now(),
	}
	if err := alert.Validate(); err != nil {
		return err
	}

	return uc.repo.CreateAlert(ctx, &alert)
}
