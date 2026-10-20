package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"time"
)

// IAddTargetRepository defines the storage operation required to add a target.
type IAddTargetRepository interface {
	Save(ctx context.Context, target *domain.Target) error
}

type AddTargetUseCase struct {
	targetRepo IAddTargetRepository
}

// NewAddTargetUseCase creates a new AddTargetUseCase with the provided repository.
func NewAddTargetUseCase(repo IAddTargetRepository) *AddTargetUseCase {
	return &AddTargetUseCase{
		targetRepo: repo,
	}
}

type AddTargetRequest struct {
	UserID         int
	Kind           string
	Address        string
	Interval       time.Duration
	Timeout        time.Duration
	ExpectedStatus int
	ExpectedBody   string
}

func (uc *AddTargetUseCase) AddTarget(ctx context.Context, req AddTargetRequest) (*domain.Target, error) {
	target := &domain.Target{
		UserID:         req.UserID,
		Kind:           req.Kind,
		Address:        req.Address,
		Interval:       req.Interval,
		Timeout:        req.Timeout,
		ExpectedStatus: req.ExpectedStatus,
		ExpectedBody:   req.ExpectedBody,
		Active:         true,
	}

	if err := target.Validate(); err != nil {
		return nil, err
	}

	if err := uc.targetRepo.Save(ctx, target); err != nil {
		return nil, err
	}

	return target, nil
}
