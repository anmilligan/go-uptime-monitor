package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
)

// IListTargetsRepository defines the storage operation required to list targets by user ID.
type IListTargetsRepository interface {
	ListByUserID(ctx context.Context, userID int) ([]domain.Target, error)
}
type ListTargetsUseCase struct {
	targetRepo IListTargetsRepository
}

func NewListTargetsUseCase(targetRepo IListTargetsRepository) *ListTargetsUseCase {
	return &ListTargetsUseCase{
		targetRepo: targetRepo,
	}
}

// ListTargets returns all targets that belong to the specified user.
func (uc *ListTargetsUseCase) ListTargets(ctx context.Context, userID int) ([]domain.Target, error) {
	if userID <= 0 {
		return nil, errors.New("user ID must be greater than zero")
	}

	return uc.targetRepo.ListByUserID(ctx, userID)
}
