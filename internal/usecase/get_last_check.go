package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
)

type ILastCheckRepository interface {
	GetLastChecks(ctx context.Context, targetID int, limit int) ([]domain.Check, error)
}

type GetLastCheckUseCase struct {
	checkRepo ILastCheckRepository
}

func NewGetLastCheckUseCase(checkRepo ILastCheckRepository) *GetLastCheckUseCase {
	return &GetLastCheckUseCase{
		checkRepo: checkRepo,
	}
}

// GetLastCheck returns the most recent check for the specified target.
// It returns nil without an error if the target has not been checked yet.
func (uc *GetLastCheckUseCase) GetLastCheck(ctx context.Context, targetID int) (*domain.Check, error) {
	if targetID <= 0 {
		return nil, errors.New("target ID must be greater than zero")
	}
	checks, err := uc.checkRepo.GetLastChecks(
		ctx,
		targetID,
		1,
	)
	if err != nil {
		return nil, err
	}

	if len(checks) == 0 {
		return nil, nil
	}

	return &checks[0], nil
}
