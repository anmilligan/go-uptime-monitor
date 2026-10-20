package usecase

import (
	"context"
	"errors"
)

// IDeleteTargetRepository defines the storage operation required to delete a target.
type IDeleteTargetRepository interface {
	Delete(ctx context.Context, id int) error
}
type DeleteTargetUseCase struct {
	targetRepo IDeleteTargetRepository
}

func NewDeleteTargetUseCase(targetRepo IDeleteTargetRepository) *DeleteTargetUseCase {
	return &DeleteTargetUseCase{
		targetRepo: targetRepo,
	}
}

// DeleteTarget deletes a target by its ID.
func (uc *DeleteTargetUseCase) DeleteTarget(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("target ID must be greater than zero")
	}

	return uc.targetRepo.Delete(ctx, id)
}
