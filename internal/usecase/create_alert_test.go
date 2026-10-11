package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAlertRepository struct {
	mock.Mock
}

func (m *MockAlertRepository) CreateAlert(
	ctx context.Context,
	alert *domain.Alert,
) (bool, error) {
	args := m.Called(ctx, alert)

	return args.Bool(0), args.Error(1)
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

type MockAlertDeduplicator struct {
	mock.Mock
}

func (m *MockAlertDeduplicator) Claim(
	ctx context.Context,
	incidentID int,
	alertType string,
	ttl time.Duration,
) (bool, error) {
	args := m.Called(
		ctx,
		incidentID,
		alertType,
		ttl,
	)

	return args.Bool(0), args.Error(1)
}

func (m *MockAlertDeduplicator) Release(
	ctx context.Context,
	incidentID int,
	alertType string,
) error {
	args := m.Called(
		ctx,
		incidentID,
		alertType,
	)

	return args.Error(0)
}

func TestCreateAlert_Success(t *testing.T) {
	// Arrange
	alertRepo := new(MockAlertRepository)
	notifier := new(MockNotifier)
	deduplicator := new(MockAlertDeduplicator)

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
		deduplicator,
	)

	message := "🔴 Website unavailable"

	deduplicator.On(
		"Claim",
		mock.Anything,
		42,
		"DOWN",
		10*time.Minute,
	).Return(true, nil).Once()

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
	).Return(true, nil).Once()

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

	deduplicator.AssertExpectations(t)
	alertRepo.AssertExpectations(t)
	notifier.AssertExpectations(t)

	deduplicator.AssertNotCalled(
		t,
		"Release",
		mock.Anything,
		42,
		"DOWN",
	)
}

func TestCreateAlert_AlreadyClaimed(t *testing.T) {
	// Arrange
	alertRepo := new(MockAlertRepository)
	notifier := new(MockNotifier)
	deduplicator := new(MockAlertDeduplicator)

	userRepo := &fakeUserRepository{}

	uc := NewCreateAlertUseCase(
		alertRepo,
		userRepo,
		notifier,
		deduplicator,
	)

	deduplicator.On(
		"Claim",
		mock.Anything,
		42,
		"DOWN",
		10*time.Minute,
	).Return(false, nil).Once()

	// Act
	err := uc.CreateAlert(
		context.Background(),
		42,
		1,
		"DOWN",
		"🔴 Website unavailable",
	)

	// Assert
	require.NoError(t, err)

	deduplicator.AssertExpectations(t)

	require.Empty(t, alertRepo.Calls)
	require.Empty(t, notifier.Calls)

	deduplicator.AssertNotCalled(
		t,
		"Release",
		mock.Anything,
		42,
		"DOWN",
	)
}

func TestCreateAlert_RepositoryError_ReleasesClaim(t *testing.T) {
	// Arrange
	alertRepo := new(MockAlertRepository)
	notifier := new(MockNotifier)
	deduplicator := new(MockAlertDeduplicator)

	userRepo := &fakeUserRepository{}

	uc := NewCreateAlertUseCase(
		alertRepo,
		userRepo,
		notifier,
		deduplicator,
	)

	expectedErr := errors.New("failed to save alert")
	message := "🔴 Website unavailable"

	deduplicator.On(
		"Claim",
		mock.Anything,
		42,
		"DOWN",
		10*time.Minute,
	).Return(true, nil).Once()

	alertRepo.On(
		"CreateAlert",
		mock.Anything,
		mock.MatchedBy(func(alert *domain.Alert) bool {
			return alert != nil &&
				alert.IncidentID == 42 &&
				alert.Type == "DOWN" &&
				alert.Message == message
		}),
	).Return(false, expectedErr).Once()

	deduplicator.On(
		"Release",
		mock.Anything,
		42,
		"DOWN",
	).Return(nil).Once()

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

	deduplicator.AssertExpectations(t)
	alertRepo.AssertExpectations(t)

	require.Empty(t, notifier.Calls)
}

func TestCreateAlert_AlreadyExistsInRepository(t *testing.T) {
	// Arrange
	alertRepo := new(MockAlertRepository)
	notifier := new(MockNotifier)
	deduplicator := new(MockAlertDeduplicator)

	userRepo := &fakeUserRepository{}

	uc := NewCreateAlertUseCase(
		alertRepo,
		userRepo,
		notifier,
		deduplicator,
	)

	message := "🔴 Website unavailable"

	deduplicator.On(
		"Claim",
		mock.Anything,
		42,
		"DOWN",
		10*time.Minute,
	).Return(true, nil).Once()

	alertRepo.On(
		"CreateAlert",
		mock.Anything,
		mock.AnythingOfType("*domain.Alert"),
	).Return(false, nil).Once()

	// Act
	err := uc.CreateAlert(
		context.Background(),
		42,
		1,
		"DOWN",
		message,
	)

	// Assert
	require.NoError(t, err)

	deduplicator.AssertExpectations(t)
	alertRepo.AssertExpectations(t)

	require.Empty(t, notifier.Calls)

	deduplicator.AssertNotCalled(
		t,
		"Release",
		mock.Anything,
		42,
		"DOWN",
	)
}

func TestCreateAlert_NotifierError_DoesNotReleaseClaim(t *testing.T) {
	// Arrange
	alertRepo := new(MockAlertRepository)
	notifier := new(MockNotifier)
	deduplicator := new(MockAlertDeduplicator)

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
		deduplicator,
	)

	message := "🔴 Website unavailable"
	expectedErr := errors.New("telegram unavailable")

	deduplicator.On(
		"Claim",
		mock.Anything,
		42,
		"DOWN",
		10*time.Minute,
	).Return(true, nil).Once()

	alertRepo.On(
		"CreateAlert",
		mock.Anything,
		mock.MatchedBy(func(alert *domain.Alert) bool {
			return alert != nil &&
				alert.IncidentID == 42 &&
				alert.Type == "DOWN" &&
				alert.Message == message
		}),
	).Return(true, nil).Once()

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

	deduplicator.AssertExpectations(t)
	alertRepo.AssertExpectations(t)
	notifier.AssertExpectations(t)

	deduplicator.AssertNotCalled(
		t,
		"Release",
		mock.Anything,
		42,
		"DOWN",
	)
}
