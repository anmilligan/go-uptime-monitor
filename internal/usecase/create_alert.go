package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
	"fmt"
	"time"
)

type IAlertRepository interface {
	CreateAlert(ctx context.Context, alert *domain.Alert) (bool, error)
}

type IAlertDeduplicator interface {
	Claim(ctx context.Context, incidentID int, alertType string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, incidentID int, alertType string) error
}

type INotifier interface {
	Send(ctx context.Context, chatID int64, message string) error
}

type CreateAlertUseCase struct {
	alertRepo    IAlertRepository
	userRepo     IUserRepository
	notifier     INotifier
	deduplicator IAlertDeduplicator
}

func NewCreateAlertUseCase(
	alertRepo IAlertRepository,
	userRepo IUserRepository,
	notifier INotifier,
	deduplicator IAlertDeduplicator,
) *CreateAlertUseCase {
	return &CreateAlertUseCase{
		alertRepo:    alertRepo,
		userRepo:     userRepo,
		notifier:     notifier,
		deduplicator: deduplicator,
	}
}

func (uc *CreateAlertUseCase) CreateAlert(
	ctx context.Context,
	incidentID int,
	userID int,
	alertType string,
	message string,
) error {
	const dedupTTL = 10 * time.Minute

	alert := &domain.Alert{
		IncidentID: incidentID,
		Type:       alertType,
		Message:    message,
		CreatedAt:  time.Now(),
	}

	if err := alert.Validate(); err != nil {
		return err
	}

	claimed, err := uc.deduplicator.Claim(
		ctx,
		incidentID,
		alertType,
		dedupTTL,
	)
	if err != nil {
		return fmt.Errorf(
			"claim alert deduplication: %w",
			err,
		)
	}

	if !claimed {
		return nil
	}

	created, err := uc.alertRepo.CreateAlert(
		ctx,
		alert,
	)
	if err != nil {
		return uc.releaseClaim(
			ctx,
			incidentID,
			alertType,
			err,
		)
	}

	if !created {
		return nil
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := uc.notifier.Send(
		ctx,
		user.TelegramID,
		message,
	); err != nil {
		return err
	}

	return nil
}

func (uc *CreateAlertUseCase) releaseClaim(
	ctx context.Context,
	incidentID int,
	alertType string,
	originalErr error,
) error {
	releaseCtx := context.WithoutCancel(ctx)

	if err := uc.deduplicator.Release(
		releaseCtx,
		incidentID,
		alertType,
	); err != nil {
		return errors.Join(
			originalErr,
			fmt.Errorf(
				"release alert deduplication: %w",
				err,
			),
		)
	}

	return originalErr
}
