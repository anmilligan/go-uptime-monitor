package scheduler

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/worker"
	"context"
	"log"
	"time"
)

type ITargetRepository interface {
	ListAll(ctx context.Context) ([]domain.Target, error)
}

type IJobQueue interface {
	Enqueue(ctx context.Context, job worker.Job) error
}

type IProcessCheckUseCase interface {
	ProcessCheck(
		ctx context.Context,
		target *domain.Target,
		result domain.CheckResult,
		consecutiveFailuresLimit int,
	) error
}

type Scheduler struct {
	targetRepo ITargetRepository
	jobQueue   IJobQueue
	nextRun    map[int]time.Time
}

func NewScheduler(targetRepo ITargetRepository, jobQueue IJobQueue) *Scheduler {
	return &Scheduler{
		targetRepo: targetRepo,
		jobQueue:   jobQueue,
		nextRun:    make(map[int]time.Time),
	}
}

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
	targets, err := s.targetRepo.ListAll(ctx)
	if err != nil {
		log.Printf("scheduler: failed to get targets: %v", err)
		return
	}

	now := time.Now()

	for i := range targets {
		target := &targets[i]

		nextRun, exists := s.nextRun[target.ID]

		if exists && now.Before(nextRun) {
			continue
		}

		job := worker.Job{
			Target: *target,
		}

		if err := s.jobQueue.Enqueue(ctx, job); err != nil {
			log.Printf(
				"scheduler: failed to enqueue target %d: %v",
				target.ID,
				err,
			)
			continue
		}

		s.nextRun[target.ID] = time.Now().Add(target.Interval)
	}
}
