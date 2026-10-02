package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
	"time"
)

type IUptimeIncidentRepository interface {
	GetIncidentsForPeriod(ctx context.Context, targetID int, from time.Time, to time.Time) ([]domain.Incident, error)
}

type UptimeResult struct {
	Percentage  float64
	ObservedFor time.Duration
	Downtime    time.Duration
	HasData     bool
}

type CalculateUptimeUseCase struct {
	incidentRepo IUptimeIncidentRepository
	now          func() time.Time
}

func NewCalculateUptimeUseCase(incidentRepo IUptimeIncidentRepository) *CalculateUptimeUseCase {
	return &CalculateUptimeUseCase{
		incidentRepo: incidentRepo,
		now:          time.Now,
	}
}

func (uc *CalculateUptimeUseCase) CalculateUptime(ctx context.Context, targetID int, targetCreatedAt time.Time, period time.Duration) (*UptimeResult, error) {
	if targetID <= 0 {
		return nil, errors.New("target ID must be greater than zero")
	}

	if period <= 0 {
		return nil, errors.New("period must be greater than zero")
	}

	now := uc.now()

	// Например, для uptime за 24 часа:
	// from = сейчас - 24 часа.
	from := now.Add(-period)

	// Но если монитор создан позже,
	// не считаем время до его создания.
	if targetCreatedAt.After(from) {
		from = targetCreatedAt
	}

	// Например, монитор был создан буквально сейчас
	// и у нас ещё нет периода наблюдения.
	if !from.Before(now) {
		return &UptimeResult{
			Percentage:  0,
			ObservedFor: 0,
			Downtime:    0,
			HasData:     false,
		}, nil
	}

	incidents, err := uc.incidentRepo.GetIncidentsForPeriod(
		ctx,
		targetID,
		from,
		now,
	)
	if err != nil {
		return nil, err
	}

	observedFor := now.Sub(from)

	var downtime time.Duration

	for _, incident := range incidents {
		incidentStart := incident.CreatedAt

		// Инцидент мог начаться раньше нашего периода.
		// Тогда считаем downtime только с начала периода.
		if incidentStart.Before(from) {
			incidentStart = from
		}

		incidentEnd := incident.ClosedAt

		// Если инцидент всё ещё OPEN,
		// downtime продолжается до текущего момента.
		if incident.Status == "OPEN" ||
			incidentEnd.IsZero() ||
			incidentEnd.After(now) {
			incidentEnd = now
		}

		// Защита от некорректного интервала.
		if !incidentEnd.After(incidentStart) {
			continue
		}

		downtime += incidentEnd.Sub(incidentStart)
	}

	// На случай некорректных/пересекающихся данных
	// не позволяем downtime стать больше всего периода наблюдения.
	if downtime > observedFor {
		downtime = observedFor
	}

	availableFor := observedFor - downtime

	percentage :=
		float64(availableFor) /
			float64(observedFor) *
			100

	return &UptimeResult{
		Percentage:  percentage,
		ObservedFor: observedFor,
		Downtime:    downtime,
		HasData:     true,
	}, nil
}
