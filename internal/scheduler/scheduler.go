package scheduler

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"log"
	"time"
)

type ITargetRepository interface {
	ListAll(ctx context.Context) ([]domain.Target, error)
}

type IChecker interface {
	Check(ctx context.Context, target *domain.Target) domain.CheckResult
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
	targetRepo               ITargetRepository
	checker                  IChecker
	processCheck             IProcessCheckUseCase
	consecutiveFailuresLimit int
	nextRun                  map[int]time.Time
}

func NewScheduler(targetRepo ITargetRepository, checker IChecker, processCheck IProcessCheckUseCase, consecutiveFailuresLimit int) *Scheduler {
	return &Scheduler{
		targetRepo:               targetRepo,
		checker:                  checker,
		processCheck:             processCheck,
		consecutiveFailuresLimit: consecutiveFailuresLimit,
		nextRun:                  make(map[int]time.Time),
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

		result := s.checker.Check(ctx, target)

		if err := s.processCheck.ProcessCheck(
			ctx,
			target,
			result,
			s.consecutiveFailuresLimit,
		); err != nil {
			log.Printf(
				"scheduler: failed to process target %d: %v",
				target.ID,
				err,
			)
		}

		s.nextRun[target.ID] = time.Now().Add(target.Interval)
	}
}
