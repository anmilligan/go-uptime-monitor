package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
)

type ILastCheckRepository interface {
	GetLastChecks(
		ctx context.Context,
		targetID int,
		limit int,
	) ([]domain.Check, error)
}

type GetLastCheckUseCase struct {
	checkRepo ILastCheckRepository
}

func NewGetLastCheckUseCase(
	checkRepo ILastCheckRepository,
) *GetLastCheckUseCase {
	return &GetLastCheckUseCase{
		checkRepo: checkRepo,
	}
}

func (uc *GetLastCheckUseCase) GetLastCheck(
	ctx context.Context,
	targetID int,
) (*domain.Check, error) {
	checks, err := uc.checkRepo.GetLastChecks(
		ctx,
		targetID,
		1,
	)
	if err != nil {
		return nil, err
	}

	// Монитор создан, но ещё ни разу не проверялся.
	if len(checks) == 0 {
		return nil, nil
	}

	return &checks[0], nil
}
