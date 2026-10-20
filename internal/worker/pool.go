package worker

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/metrics"
	"context"
	"log/slog"
	"sync"
)

// IChecker defines the operation required to check a target.
type IChecker interface {
	Check(ctx context.Context, target *domain.Target) domain.CheckResult
}

// IProcessCheckUseCase defines the business operation required
// to process and persist a completed check.
type IProcessCheckUseCase interface {
	ProcessCheck(ctx context.Context, target *domain.Target, result domain.CheckResult, consecutiveFailuresLimit int) error
}

// Pool manages a fixed set of workers that process queued check jobs.
type Pool struct {
	jobs                     chan Job
	checker                  IChecker
	processCheck             IProcessCheckUseCase
	workerCount              int
	consecutiveFailuresLimit int

	wg sync.WaitGroup
}

// NewPool creates a worker pool with the specified concurrency
// and failure threshold.
func NewPool(workerCount int, checker IChecker, processCheck IProcessCheckUseCase, consecutiveFailuresLimit int) *Pool {
	return &Pool{
		jobs:                     make(chan Job, workerCount),
		checker:                  checker,
		processCheck:             processCheck,
		workerCount:              workerCount,
		consecutiveFailuresLimit: consecutiveFailuresLimit,
	}
}

// Start launches the configured number of workers.
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)

		go func(workerID int) {
			defer p.wg.Done()

			p.runWorker(ctx, workerID)
		}(i + 1)
	}
}

// runWorker continuously processes jobs from the queue until the context is canceled.
// For each job, it checks the target, records metrics, and processes the check result.
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

// Enqueue adds a check job to the worker queue or returns
// when the context is canceled.
func (p *Pool) Enqueue(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	case p.jobs <- job:
		return nil
	}
}

// Wait waits until all workers stop or the provided context expires.
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
