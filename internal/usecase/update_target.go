package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
)

type IUpdateTargetRepository interface {
	Update(ctx context.Context, target *domain.Target) error
}

type UpdateTargetUseCase struct {
	targetRepo IUpdateTargetRepository
}

func NewUpdateTargetUseCase(targetRepo IUpdateTargetRepository) *UpdateTargetUseCase {
	return &UpdateTargetUseCase{
		targetRepo: targetRepo,
	}
}

func (uc *UpdateTargetUseCase) UpdateTarget(ctx context.Context, target *domain.Target) error {
	if err := target.Validate(); err != nil {
		return err
	}

	return uc.targetRepo.Update(ctx, target)
}
