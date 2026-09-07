package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
)

type ITargetRepository interface {
	IAddTargetRepository
	IGetTargetRepository
	IListTargetsRepository
	IDeleteTargetRepository
	IUpdateTargetRepository
	ListAll(ctx context.Context) ([]domain.Target, error)
}
