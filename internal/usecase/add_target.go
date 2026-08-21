package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"
)

type TargetRepository interface {
	Save(ctx context.Context, target *domain.Target) error
}

type AddTargetUseCase struct { // сценарий в котором лежит инструмент для работы с БД
	targetRepo TargetRepository
}

func NewAddTargetUseCase(repo TargetRepository) *AddTargetUseCase { // функция которая создает наш сценарий
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

func (uc *AddTargetUseCase) Execute(ctx context.Context, req AddTargetRequest) (*domain.Target, error) { // инструкция по добавлению сайта
	target := &domain.Target{
		UserID:         req.UserID,
		Kind:           req.Kind,
		Interval:       req.Interval,
		Timeout:        req.Timeout,
		ExpectedStatus: req.ExpectedStatus,
		ExpectedBody:   req.ExpectedBody,
		CreatedAt:      time.Now(), // время создания ставим автоматически
	}

	if err := target.Validate(); err != nil { // запускаем проверку написанную в domain
		return nil, err
	}
	if err := uc.targetRepo.Save(ctx, target); err != nil { // если все в порядке - просим репозиторий сохранить сайт в БД
		return nil, err
	}
	return target, nil // вернув созданный таргет появляется ID от БД
}
