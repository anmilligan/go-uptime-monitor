package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
)

// IGetTargetRepository defines the storage operation required to retrieve a target by ID.
type IGetTargetRepository interface {
	GetByID(ctx context.Context, id int) (*domain.Target, error)
}

// GetTargetUseCase handles retrieving a target by its ID.
type GetTargetUseCase struct {
	targetRepo IGetTargetRepository
}

func NewGetTargetUseCase(targetRepo IGetTargetRepository) *GetTargetUseCase {
	return &GetTargetUseCase{
		targetRepo: targetRepo,
	}
}

func (uc *GetTargetUseCase) GetTarget(ctx context.Context, id int) (*domain.Target, error) {
	if id <= 0 {
		return nil, errors.New("target ID must be greater than zero")
	}

	return uc.targetRepo.GetByID(ctx, id)
}
