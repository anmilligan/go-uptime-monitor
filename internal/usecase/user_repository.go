package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
)

type IUserRepository interface {
	GetByID(ctx context.Context, id int) (*domain.User, error)
}
