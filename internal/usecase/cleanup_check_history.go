package usecase

import (
	"context"
	"errors"
	"time"
)

// ICheckHistoryCleanupRepository defines the storage operation
// required to delete checks older than a specified time.
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

// Cleanup deletes checks older than the configured retention period
// and returns the number of deleted records.
func (uc *CleanupCheckHistoryUseCase) Cleanup(ctx context.Context) (int64, error) {
	if uc.retention <= 0 {
		return 0, errors.New("retention must be positive")
	}

	cutoff := time.Now().Add(-uc.retention)

	return uc.repo.DeleteChecksBefore(
		ctx,
		cutoff,
	)
}
