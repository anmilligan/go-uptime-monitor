package usecase

import (
	"context"
	"fmt"
	"time"
)

type ICheckHistoryCleanupRepository interface {
	DeleteChecksBefore(ctx context.Context, before time.Time) (int64, error)
}

type CleanupCheckHistoryUseCase struct {
	repo      ICheckHistoryCleanupRepository
	retention time.Duration
}

func NewCleanupCheckHistoryUseCase(repo ICheckHistoryCleanupRepository, retention time.Duration) *CleanupCheckHistoryUseCase {
	return &CleanupCheckHistoryUseCase{
		repo:      repo,
		retention: retention,
	}
}

func (uc *CleanupCheckHistoryUseCase) Cleanup(ctx context.Context) (int64, error) {
	if uc.retention <= 0 {
		return 0, fmt.Errorf("retention must be positive")
	}

	cutoff := time.Now().Add(-uc.retention)

	return uc.repo.DeleteChecksBefore(
		ctx,
		cutoff,
	)
}
