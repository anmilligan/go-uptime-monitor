package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAlertRepository struct {
	mock.Mock
}

func (m *MockAlertRepository) CreateAlert(
	ctx context.Context,
	alert *domain.Alert,
) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

type MockNotifier struct {
	mock.Mock
}

func (m *MockNotifier) Send(
	ctx context.Context,
	chatID int64,
	message string,
) error {
	args := m.Called(ctx, chatID, message)
	return args.Error(0)
}

func TestCreateAlert_Success(t *testing.T) {
	// Arrange
	alertRepo := new(MockAlertRepository)
	notifier := new(MockNotifier)

	user := &domain.User{
		ID:         1,
		TelegramID: 123456789,
	}

	userRepo := &fakeUserRepository{
		getUser: user,
	}

	uc := NewCreateAlertUseCase(
		alertRepo,
		userRepo,
		notifier,
	)

	message := "🔴 Website unavailable"

	// Проверяем, что сначала сохраняется нужный алерт.
	alertRepo.On(
		"CreateAlert",
		mock.Anything,
		mock.MatchedBy(func(alert *domain.Alert) bool {
			return alert != nil &&
				alert.IncidentID == 42 &&
				alert.Type == "DOWN" &&
				alert.Message == message &&
				!alert.CreatedAt.IsZero()
		}),
	).Return(nil).Once()

	// После сохранения сообщение отправляется пользователю.
	notifier.On(
		"Send",
		mock.Anything,
		user.TelegramID,
		message,
	).Return(nil).Once()

	// Act
	err := uc.CreateAlert(
		context.Background(),
		42,
		user.ID,
		"DOWN",
		message,
	)

	// Assert
	require.NoError(t, err)

	alertRepo.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestCreateAlert_RepositoryError(t *testing.T) {
	// Arrange
	alertRepo := new(MockAlertRepository)
	notifier := new(MockNotifier)
	userRepo := &fakeUserRepository{}

	uc := NewCreateAlertUseCase(
		alertRepo,
		userRepo,
		notifier,
	)

	expectedErr := errors.New("failed to save alert")
	message := "🔴 Website unavailable"

	alertRepo.On(
		"CreateAlert",
		mock.Anything,
		mock.MatchedBy(func(alert *domain.Alert) bool {
			return alert != nil &&
				alert.IncidentID == 42 &&
				alert.Type == "DOWN" &&
				alert.Message == message
		}),
	).Return(expectedErr).Once()

	// Act
	err := uc.CreateAlert(
		context.Background(),
		42,
		1,
		"DOWN",
		message,
	)

	// Assert
	require.ErrorIs(t, err, expectedErr)

	alertRepo.AssertExpectations(t)

	// При ошибке сохранения Telegram не вызывается.
	require.Empty(t, notifier.Calls)
}

func TestCreateAlert_NotifierError(t *testing.T) {
	// Arrange
	alertRepo := new(MockAlertRepository)
	notifier := new(MockNotifier)

	user := &domain.User{
		ID:         1,
		TelegramID: 123456789,
	}

	userRepo := &fakeUserRepository{
		getUser: user,
	}

	uc := NewCreateAlertUseCase(
		alertRepo,
		userRepo,
		notifier,
	)

	message := "🔴 Website unavailable"
	expectedErr := errors.New("telegram unavailable")

	// Сохранение алерта проходит успешно.
	alertRepo.On(
		"CreateAlert",
		mock.Anything,
		mock.MatchedBy(func(alert *domain.Alert) bool {
			return alert != nil &&
				alert.IncidentID == 42 &&
				alert.Type == "DOWN" &&
				alert.Message == message
		}),
	).Return(nil).Once()

	// Но Telegram возвращает ошибку.
	notifier.On(
		"Send",
		mock.Anything,
		user.TelegramID,
		message,
	).Return(expectedErr).Once()

	// Act
	err := uc.CreateAlert(
		context.Background(),
		42,
		user.ID,
		"DOWN",
		message,
	)

	// Assert
	require.ErrorIs(t, err, expectedErr)

	alertRepo.AssertExpectations(t)
	notifier.AssertExpectations(t)
}
