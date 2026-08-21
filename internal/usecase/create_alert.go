package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"
)

type AlertRepository interface {
	CreateAlert(ctx context.Context, alert *domain.Alert) error
}

type CreateAlertUseCase struct {
	repo AlertRepository
}

func NewCreateAlertUseCase(repo AlertRepository) *CreateAlertUseCase {
	return &CreateAlertUseCase{
		repo: repo,
	}
}

func (uc *CreateAlertUseCase) Execute(ctx context.Context, targetID int, alertType string, message string) error {
	alert := domain.Alert{
		TargetID:  targetID,
		Type:      alertType,
		Message:   message,
		CreatedAt: time.Now(),
	}
	if err := alert.Validate(); err != nil {
		return err
	}

	return uc.repo.CreateAlert(ctx, &alert)
}
