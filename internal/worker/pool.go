package worker

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/metrics"
	"context"
	"log/slog"
	"sync"
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

	wg sync.WaitGroup
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
		p.wg.Add(1)

		go func(workerID int) {
			defer p.wg.Done()

			p.runWorker(ctx, workerID)
		}(i + 1)
	}
}

func (p *Pool) runWorker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return

		case job := <-p.jobs:
			if ctx.Err() != nil {
				return
			}

			target := job.Target

			jobCtx := context.WithoutCancel(ctx)

			result := p.checker.Check(
				jobCtx,
				&target,
			)

			metrics.RecordCheck()

			metrics.RecordCheckLatency(
				target.ID,
				result.ResponseTime,
			)

			if err := p.processCheck.ProcessCheck(
				jobCtx,
				&target,
				result,
				p.consecutiveFailuresLimit,
			); err != nil {
				slog.ErrorContext(
					jobCtx,
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

func (p *Pool) Wait(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}
