package scheduler

import (
	"context"
	"log/slog"
	"time"
)

// ICheckHistoryCleanupUseCase defines the operation required
// to clean up expired check history.
type ICheckHistoryCleanupUseCase interface {
	Cleanup(ctx context.Context) (int64, error)
}

type CleanupScheduler struct {
	cleanup  ICheckHistoryCleanupUseCase
	interval time.Duration
}

func NewCleanupScheduler(cleanup ICheckHistoryCleanupUseCase, interval time.Duration) *CleanupScheduler {
	return &CleanupScheduler{
		cleanup:  cleanup,
		interval: interval,
	}
}

// Run starts the cleanup loop and stops when the context is canceled.
func (s *CleanupScheduler) Run(ctx context.Context) {
	s.runCleanup(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.runCleanup(ctx)
		}
	}
}

// runCleanup executes one cleanup cycle and logs the result.
func (s *CleanupScheduler) runCleanup(ctx context.Context) {
	deleted, err := s.cleanup.Cleanup(ctx)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to clean check history",
			"error", err,
		)
		return
	}

	slog.InfoContext(
		ctx,
		"check history cleanup completed",
		"deleted_checks", deleted,
	)
}
