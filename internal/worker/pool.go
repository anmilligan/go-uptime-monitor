package worker

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/metrics"
	"context"
	"log/slog"
)

type IChecker interface {
	Check(ctx context.Context, target *domain.Target) domain.CheckResult
}

type IProcessCheckUseCase interface {
	ProcessCheck(ctx context.Context, target *domain.Target, result domain.CheckResult, consecutiveFailuresLimit int) error
}
type Pool struct {
	jobs                     chan Job
	checker                  IChecker
	processCheck             IProcessCheckUseCase
	workerCount              int
	consecutiveFailuresLimit int
}

func NewPool(
	workerCount int,
	checker IChecker,
	processCheck IProcessCheckUseCase,
	consecutiveFailuresLimit int,
) *Pool {
	return &Pool{
		jobs:                     make(chan Job, workerCount),
		checker:                  checker,
		processCheck:             processCheck,
		workerCount:              workerCount,
		consecutiveFailuresLimit: consecutiveFailuresLimit,
	}
}
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		go p.runWorker(ctx, i+1)
	}
}

func (p *Pool) runWorker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return

		case job := <-p.jobs:
			target := job.Target

			result := p.checker.Check(ctx, &target)

			metrics.RecordCheck()

			metrics.RecordCheckLatency(target.ID, result.ResponseTime)

			if err := p.processCheck.ProcessCheck(
				ctx,
				&target,
				result,
				p.consecutiveFailuresLimit,
			); err != nil {
				slog.ErrorContext(
					ctx,
					"worker failed to process monitor",
					"worker_id", workerID,
					"monitor_id", target.ID,
					"error", err,
				)
			}
		}
	}
}

func (p *Pool) Enqueue(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	case p.jobs <- job:
		return nil
	}
}
