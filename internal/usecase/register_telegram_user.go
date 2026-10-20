package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"fmt"
	"strings"
)

type RegisterTelegramUserUseCase struct {
	userRepo IUserRepository
}

func NewRegisterTelegramUserUseCase(userRepo IUserRepository) *RegisterTelegramUserUseCase {
	return &RegisterTelegramUserUseCase{
		userRepo: userRepo,
	}
}

// Register registers or updates a Telegram user in the application.
func (uc *RegisterTelegramUserUseCase) Register(ctx context.Context, telegramID int64, username string) (*domain.User, error) {
	username = strings.TrimSpace(username)

	if username == "" {
		username = fmt.Sprintf("tg_%d", telegramID)
	}

	user := &domain.User{
		TelegramID: telegramID,
		Username:   username,
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	if err := uc.userRepo.Upsert(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
