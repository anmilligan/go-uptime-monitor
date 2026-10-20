package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
)

// IUpdateTargetRepository defines the storage operation required to update a target.
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

// UpdateTarget validates and updates an existing target.
func (uc *UpdateTargetUseCase) UpdateTarget(ctx context.Context, target *domain.Target) error {
	if target == nil {
		return errors.New("target cannot be nil")
	}

	if target.ID <= 0 {
		return errors.New("target ID must be greater than zero")
	}

	if err := target.Validate(); err != nil {
		return err
	}

	return uc.targetRepo.Update(ctx, target)
}
