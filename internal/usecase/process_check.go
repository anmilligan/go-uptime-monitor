package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"
)

type CheckAndIncidentRepo interface { // интерфейс со списком требований к БД
	SaveCheck(ctx context.Context, check *domain.Check) error
	GetLastChecks(ctx context.Context, targetID int, limit int) ([]domain.Check, error)
	GetOpenIncident(ctx context.Context, targetID int) (*domain.Incident, error)
	CreateIncident(ctx context.Context, incident *domain.Incident) error
	CloseIncident(ctx context.Context, incidentID int, closedAt time.Time) error
}

type AlertCreator interface {
	Execute(ctx context.Context, targetID int, alertType string, message string) error
}

type ProcessCheckUseCase struct {
	repo  CheckAndIncidentRepo
	alert AlertCreator
}

func NewProcessCheckUseCase(repo CheckAndIncidentRepo, alert AlertCreator) *ProcessCheckUseCase {
	return &ProcessCheckUseCase{
		repo:  repo,
		alert: alert,
	}
}

func (uc *ProcessCheckUseCase) Execute(ctx context.Context, target *domain.Target, result CheckResult, consecutiveFailuresLimit int) error {
	// создаем и сохраняем технический лог проверки
	check := &domain.Check{
		TargetID:     target.ID,
		Status:       result.Status,
		Answer:       result.Answer,
		ResponseTime: result.ResponseTime,
		CheckedAt:    result.CheckedAt,
		IsSuccess:    result.IsSuccess,
	}
	if err := uc.repo.SaveCheck(ctx, check); err != nil {
		return err
	}

	if !result.IsSuccess {
		return uc.handleFailure(ctx, target, consecutiveFailuresLimit)
	}

	return uc.handleRecovery(ctx, target)
}

func (uc *ProcessCheckUseCase) handleFailure(ctx context.Context, target *domain.Target, limit int) error {
	existing, err := uc.repo.GetOpenIncident(ctx, target.ID)
	// проверяем не открыт ли инцидент уже
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	history, err := uc.repo.GetLastChecks(ctx, target.ID, limit) // запрашиваем историю последних проверок, чтобы узнать количество подряд идущих ошибок
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

	if err := uc.repo.CreateIncident(ctx, incident); err != nil {
		return err
	}

	return uc.alert.Execute( // отправляем событие создания alert
		ctx,
		target.ID,
		"DOWN",
		"🔴 Website unavailable",
	)
}

func (uc *ProcessCheckUseCase) handleRecovery(ctx context.Context, target *domain.Target) error { // создаем метод восстановления
	openIncident, err := uc.repo.GetOpenIncident(ctx, target.ID) // получаем открытый инцидент
	if err != nil {
		return err
	}
	if openIncident == nil {
		return nil
	}
	closedAt := time.Now()

	if err := uc.repo.CloseIncident(ctx, openIncident.ID, closedAt); err != nil { // закрываем инцидент
		return err
	}

	return uc.alert.Execute( // создали alert
		ctx,
		target.ID,
		"UP",
		"🟢 Website is available again",
	)
}
