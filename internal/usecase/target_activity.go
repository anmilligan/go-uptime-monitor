package usecase

import (
	"context"
	"errors"
)

// ITargetActivityRepository defines storage operations for changing target activity.
type ITargetActivityRepository interface {
	Pause(ctx context.Context, id int) error
	Resume(ctx context.Context, id int) error
}

type PauseTargetUseCase struct {
	targetRepo ITargetActivityRepository
}

func NewPauseTargetUseCase(targetRepo ITargetActivityRepository) *PauseTargetUseCase {
	return &PauseTargetUseCase{
		targetRepo: targetRepo,
	}
}

func (uc *PauseTargetUseCase) PauseTarget(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("target ID must be greater than zero")
	}

	return uc.targetRepo.Pause(ctx, id)
}

type ResumeTargetUseCase struct {
	targetRepo ITargetActivityRepository
}

func NewResumeTargetUseCase(targetRepo ITargetActivityRepository) *ResumeTargetUseCase {
	return &ResumeTargetUseCase{
		targetRepo: targetRepo,
	}
}

func (uc *ResumeTargetUseCase) ResumeTarget(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("target ID must be greater than zero")
	}

	return uc.targetRepo.Resume(ctx, id)
}
