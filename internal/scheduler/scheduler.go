package scheduler

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/metrics"
	"Uptime_Monitor/internal/worker"
	"context"
	"log/slog"
	"time"
)

// ITargetRepository defines the operation required to claim targets
// that are due for scheduled checks.
type ITargetRepository interface {
	ClaimDueTargets(ctx context.Context, limit int) ([]domain.ScheduledTarget, error)
}

// IJobQueue defines the operation required to enqueue check jobs.
type IJobQueue interface {
	Enqueue(ctx context.Context, job worker.Job) error
}

// Scheduler periodically claims due targets and enqueues them for checking.
type Scheduler struct {
	targetRepo ITargetRepository
	jobQueue   IJobQueue
	batchSize  int
}

func NewScheduler(targetRepo ITargetRepository, jobQueue IJobQueue, batchSize int) *Scheduler {
	return &Scheduler{
		targetRepo: targetRepo,
		jobQueue:   jobQueue,
		batchSize:  batchSize,
	}
}

// Run starts the scheduling loop and stops when the context is canceled.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	s.runCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.runCycle(ctx)
		}
	}
}

func (s *Scheduler) runCycle(ctx context.Context) {
	targets, err := s.targetRepo.ClaimDueTargets(
		ctx,
		s.batchSize,
	)
	if err != nil {
		if ctx.Err() != nil {
			return
		}

		slog.ErrorContext(
			ctx,
			"scheduler failed to claim due targets",
			"error", err,
		)
		return
	}

	for i := range targets {
		if ctx.Err() != nil {
			return
		}

		scheduled := targets[i]

		job := worker.Job{
			Target: scheduled.Target,
		}

		if err := s.jobQueue.Enqueue(ctx, job); err != nil {
			if ctx.Err() != nil {
				return
			}

			slog.ErrorContext(
				ctx,
				"scheduler failed to enqueue target",
				"monitor_id", scheduled.Target.ID,
				"error", err,
			)

			continue
		}

		lag := time.Since(scheduled.ScheduledAt)

		metrics.RecordSchedulerLag(lag)
	}
}
