package core

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
	"time"
)

type CheckAndIncidentRepo interface { // интерфейс со списком требований к БД
	SaveCheck(ctx context.Context, check *domain.Check) error
	GetLastChecks(ctx context.Context, targetID int, limit int) ([]domain.Check, error)
	GetOpenIncident(ctx context.Context, targetID int) (*domain.Incident, error)
	CreateIncident(ctx context.Context, incident *domain.Incident) error
	CloseIncident(ctx context.Context, incidentID int, closedAt time.Time) error
	CreateAlert(ctx context.Context, alert *domain.Alert) error
}

type ProcessCheckUseCase struct {
	repo CheckAndIncidentRepo
}

func NewProcessCheckUseCase(repo CheckAndIncidentRepo) *ProcessCheckUseCase {
	return &ProcessCheckUseCase{
		repo: repo,
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
	existing, err := uc.repo.GetOpenIncident(ctx, target.ID) // проверяем не открыт ли инцидент уже
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
		if limit <= 0 {
			return errors.New("лимит последовательных ошибок должен быть больше нуля")
		}
	}

	incident := &domain.Incident{
		TargetID:  target.ID,
		Status:    "OPEN",
		CreatedAt: time.Now(),
	}
	return uc.repo.CreateIncident(ctx, incident)
}

func (uc *ProcessCheckUseCase) handleRecovery(ctx context.Context, target *domain.Target) error { // создаем метод восстановления
	openIncident, err := uc.repo.GetOpenIncident(ctx, target.ID) // ищем активный инцидент
	if err != nil {
		return err
	}
	if openIncident == nil {
		return nil
	}
	closedAt := time.Now()

	return uc.repo.CloseIncident(ctx, openIncident.ID, closedAt) // отправляем в базу данных команду обновить запись, чтобы сохранить время закрытия и статус инцидента
}
