package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
)

type IGetTargetRepository interface {
	GetByID(ctx context.Context, id int) (*domain.Target, error)
}

type GetTargetUseCase struct {
	targetRepo IGetTargetRepository
}

func NewGetTargetUseCase(targetRepo IGetTargetRepository) *GetTargetUseCase {
	return &GetTargetUseCase{
		targetRepo: targetRepo}
}

func (uc *GetTargetUseCase) GetTarget(ctx context.Context, id int) (*domain.Target, error) {
	return uc.targetRepo.GetByID(ctx, id)
}
