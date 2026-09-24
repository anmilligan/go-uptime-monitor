package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"
)

type IAlertRepository interface {
	CreateAlert(ctx context.Context, alert *domain.Alert) error
}

type INotifier interface {
	Send(ctx context.Context, chatID int64, message string) error
}

type CreateAlertUseCase struct {
	alertRepo IAlertRepository
	userRepo  IUserRepository
	notifier  INotifier
}

func NewCreateAlertUseCase(
	alertRepo IAlertRepository,
	userRepo IUserRepository,
	notifier INotifier,
) *CreateAlertUseCase {
	return &CreateAlertUseCase{
		alertRepo: alertRepo,
		userRepo:  userRepo,
		notifier:  notifier,
	}
}

func (uc *CreateAlertUseCase) CreateAlert(
	ctx context.Context,
	incidentID int,
	userID int,
	alertType string,
	message string,
) error {
	alert := &domain.Alert{
		IncidentID: incidentID,
		Type:       alertType,
		Message:    message,
		CreatedAt:  time.Now(),
	}

	if err := alert.Validate(); err != nil {
		return err
	}

	if err := uc.alertRepo.CreateAlert(ctx, alert); err != nil {
		return err
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	return uc.notifier.Send(ctx, user.TelegramID, message)
}
