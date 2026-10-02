package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
)

type IUserRepository interface {
	GetByID(ctx context.Context, id int) (*domain.User, error)
	Upsert(ctx context.Context, user *domain.User) error
}
