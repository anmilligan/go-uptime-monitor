package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
)

type ICheckHistoryRepository interface {
	GetLastChecks(ctx context.Context, targetID int, limit int) ([]domain.Check, error)
}

type GetCheckHistoryUseCase struct {
	checkRepo ICheckHistoryRepository
}

func NewGetCheckHistoryUseCase(checkRepo ICheckHistoryRepository) *GetCheckHistoryUseCase {
	return &GetCheckHistoryUseCase{
		checkRepo: checkRepo,
	}
}

func (uc *GetCheckHistoryUseCase) GetCheckHistory(ctx context.Context, targetID int, limit int) ([]domain.Check, error) {
	if targetID <= 0 {
		return nil, errors.New(
			"target ID must be greater than zero",
		)
	}

	if limit <= 0 {
		return nil, errors.New(
			"limit must be greater than zero",
		)
	}

	return uc.checkRepo.GetLastChecks(
		ctx,
		targetID,
		limit,
	)
}
