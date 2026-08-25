package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"
)

type ITargetRepository interface {
	Save(ctx context.Context, target *domain.Target) error
}

type AddTargetUseCase struct { // сценарий в котором лежит инструмент для работы с БД
	targetRepo ITargetRepository
}

func NewAddTargetUseCase(repo ITargetRepository) *AddTargetUseCase { // функция которая создает наш сценарий
	return &AddTargetUseCase{
		targetRepo: repo,
	}
}

type AddTargetRequest struct { // параметры описываемые пользователем
	UserID         int
	Kind           string
	Address        string
	Interval       time.Duration
	Timeout        time.Duration
	ExpectedStatus int
	ExpectedBody   string
}

func (uc *AddTargetUseCase) AddTarget(ctx context.Context, req AddTargetRequest) (*domain.Target, error) { // инструкция по добавлению сайта
	target := &domain.Target{
		UserID:         req.UserID,
		Kind:           req.Kind,
		Address:        req.Address,
		Interval:       req.Interval,
		Timeout:        req.Timeout,
		ExpectedStatus: req.ExpectedStatus,
		ExpectedBody:   req.ExpectedBody,
	}

	if err := target.Validate(); err != nil { // запускаем проверку написанную в domain
		return nil, err
	}
	if err := uc.targetRepo.Save(ctx, target); err != nil { // если все в порядке - просим репозиторий сохранить сайт в БД
		return nil, err
	}
	return target, nil // вернув созданный таргет появляется ID от БД
}
