package usecase

import "context"

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

func (uc *DeleteTargetUseCase) DeleteTarget(ctx context.Context, id int) error {
	return uc.targetRepo.Delete(ctx, id)
}
