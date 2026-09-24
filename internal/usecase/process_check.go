package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"
)

type ICheckRepository interface {
	SaveCheck(ctx context.Context, check *domain.Check) error
	GetLastChecks(ctx context.Context, targetID int, limit int) ([]domain.Check, error)
}

type IIncidentRepository interface {
	GetOpenIncident(ctx context.Context, targetID int) (*domain.Incident, error)
	CreateIncident(ctx context.Context, incident *domain.Incident) error
	CloseIncident(ctx context.Context, incidentID int, closedAt time.Time) error
}

type IAlertCreator interface {
	CreateAlert(ctx context.Context, incidentID int, userID int, alertType string, message string) error
}

type ProcessCheckUseCase struct {
	checkRepo    ICheckRepository
	incidentRepo IIncidentRepository
	alert        IAlertCreator
}

func NewProcessCheckUseCase(checkRepo ICheckRepository, incidentRepo IIncidentRepository, alert IAlertCreator) *ProcessCheckUseCase {
	return &ProcessCheckUseCase{
		checkRepo:    checkRepo,
		incidentRepo: incidentRepo,
		alert:        alert,
	}
}

func (uc *ProcessCheckUseCase) ProcessCheck(ctx context.Context, target *domain.Target, result domain.CheckResult, consecutiveFailuresLimit int) error {
	// создаем и сохраняем технический лог проверки
	check := &domain.Check{
		TargetID:     target.ID,
		Status:       result.Status,
		Answer:       result.Answer,
		ResponseTime: result.ResponseTime,
		CheckedAt:    result.CheckedAt,
		IsSuccess:    result.IsSuccess,
	}

	if err := check.Validate(); err != nil {
		return err
	}
	if err := uc.checkRepo.SaveCheck(ctx, check); err != nil {
		return err
	}

	if !result.IsSuccess {
		return uc.handleFailure(ctx, target, consecutiveFailuresLimit)
	}

	return uc.handleRecovery(ctx, target)
}

func (uc *ProcessCheckUseCase) handleFailure(ctx context.Context, target *domain.Target, limit int) error {
	existing, err := uc.incidentRepo.GetOpenIncident(ctx, target.ID)
	// проверяем не открыт ли инцидент уже
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	history, err := uc.checkRepo.GetLastChecks(ctx, target.ID, limit) // запрашиваем историю последних проверок, чтобы узнать количество подряд идущих ошибок
	if err != nil {
		return err
	}

	failuresCount := 0 // считаем, сколько последних проверок были неудачными

	for _, check := range history {
		if !check.IsSuccess {
			failuresCount++ // увеличиваем счетчик ошибок
		} else {
			break // наткнулись на успешную проверку — серия прервалась
		}

	}

	if failuresCount < limit { // недостаточно ошибок подряд, чтобы вернуть инцидент
		return nil
	}

	incident := &domain.Incident{
		TargetID:  target.ID,
		Status:    "OPEN",
		CreatedAt: time.Now(),
	}

	if err := incident.Validate(); err != nil {
		return err
	}

	if err := uc.incidentRepo.CreateIncident(ctx, incident); err != nil {
		return err
	}

	return uc.alert.CreateAlert(
		ctx,
		incident.ID,
		target.UserID,
		"DOWN",
		"🔴 Website unavailable",
	)
}

func (uc *ProcessCheckUseCase) handleRecovery(ctx context.Context, target *domain.Target) error { // создаем метод восстановления
	openIncident, err := uc.incidentRepo.GetOpenIncident(ctx, target.ID) // получаем открытый инцидент
	if err != nil {
		return err
	}
	if openIncident == nil {
		return nil
	}
	closedAt := time.Now()

	if err := uc.incidentRepo.CloseIncident(ctx, openIncident.ID, closedAt); err != nil { // закрываем инцидент
		return err
	}

	return uc.alert.CreateAlert(
		ctx,
		openIncident.ID,
		target.UserID,
		"UP",
		"🟢 Website is available again",
	)
}
