package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
)

type IListTargetsRepository interface {
	ListByUserID(cnt context.Context, userID int) ([]domain.Target, error)
}

type ListTargetsUseCase struct {
	targetRepo IListTargetsRepository
}

func NewListTargetsUseCase(targetRepo IListTargetsRepository) *ListTargetsUseCase {
	return &ListTargetsUseCase{
		targetRepo: targetRepo,
	}
}
func (uc *ListTargetsUseCase) ListTargets(ctx context.Context, userID int) ([]domain.Target, error) {
	return uc.targetRepo.ListByUserID(ctx, userID)
}
