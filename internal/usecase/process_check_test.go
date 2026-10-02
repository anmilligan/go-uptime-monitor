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

// Мок репозитория проверок.
type MockCheckRepository struct {
	mock.Mock
}

func (m *MockCheckRepository) SaveCheck(
	ctx context.Context,
	check *domain.Check,
) error {
	args := m.Called(ctx, check)
	return args.Error(0)
}

func (m *MockCheckRepository) GetLastChecks(
	ctx context.Context,
	targetID int,
	limit int,
) ([]domain.Check, error) {
	args := m.Called(ctx, targetID, limit)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]domain.Check), args.Error(1)
}

// Мок репозитория инцидентов.
type MockIncidentRepository struct {
	mock.Mock
}

func (m *MockIncidentRepository) GetOpenIncident(
	ctx context.Context,
	targetID int,
) (*domain.Incident, error) {
	args := m.Called(ctx, targetID)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.Incident), args.Error(1)
}

func (m *MockIncidentRepository) CreateIncident(
	ctx context.Context,
	incident *domain.Incident,
) error {
	args := m.Called(ctx, incident)
	return args.Error(0)
}

func (m *MockIncidentRepository) CloseIncident(
	ctx context.Context,
	incidentID int,
	closedAt time.Time,
) error {
	args := m.Called(ctx, incidentID, closedAt)
	return args.Error(0)
}

// Мок создания алертов.
type MockAlertCreator struct {
	mock.Mock
}

func (m *MockAlertCreator) CreateAlert(
	ctx context.Context,
	incidentID int,
	userID int,
	alertType string,
	message string,
) error {
	args := m.Called(
		ctx,
		incidentID,
		userID,
		alertType,
		message,
	)

	return args.Error(0)
}

func TestProcessCheck_Success(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     1,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       200,
		Answer:       "OK",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    true,
	}

	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.MatchedBy(func(check *domain.Check) bool {
			return check != nil &&
				check.TargetID == target.ID &&
				check.Status == 200 &&
				check.IsSuccess
		}),
	).Return(nil).Once()

	// Открытого инцидента для сайта нет.
	incidentRepo.On(
		"GetOpenIncident",
		mock.Anything,
		target.ID,
	).Return(nil, nil).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.NoError(t, err)

	checkRepo.AssertExpectations(t)
	incidentRepo.AssertExpectations(t)

	incidentRepo.AssertNotCalled(
		t,
		"CreateIncident",
		mock.Anything,
		mock.Anything,
	)

	alertCreator.AssertNotCalled(
		t,
		"CreateAlert",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestProcessCheck_BelowFailureThreshold(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     1,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       200,
		Answer:       "Unexpected status",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    false,
	}

	// Текущая неудачная проверка сохраняется.
	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.MatchedBy(func(check *domain.Check) bool {
			return check != nil &&
				check.TargetID == target.ID &&
				!check.IsSuccess
		}),
	).Return(nil).Once()

	// Открытого инцидента пока нет.
	incidentRepo.On(
		"GetOpenIncident",
		mock.Anything,
		target.ID,
	).Return(nil, nil).Once()

	// Имитируем две последние неудачные проверки,
	// включая ту, которую только что сохранили.
	history := []domain.Check{
		{TargetID: 1, IsSuccess: false},
		{TargetID: 1, IsSuccess: false},
	}

	checkRepo.On(
		"GetLastChecks",
		mock.Anything,
		target.ID,
		3,
	).Return(history, nil).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.NoError(t, err)

	checkRepo.AssertExpectations(t)
	incidentRepo.AssertExpectations(t)

	incidentRepo.AssertNumberOfCalls(
		t,
		"CreateIncident",
		0,
	)

	alertCreator.AssertNumberOfCalls(
		t,
		"CreateAlert",
		0,
	)
}

func TestProcessCheck_OpensIncident(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     1,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       200,
		Answer:       "Unexpected status",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    false,
	}

	// Текущая проверка должна сохраниться.
	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.MatchedBy(func(check *domain.Check) bool {
			return check != nil &&
				check.TargetID == target.ID &&
				!check.IsSuccess
		}),
	).Return(nil).Once()

	// Открытого инцидента ещё нет.
	incidentRepo.On(
		"GetOpenIncident",
		mock.Anything,
		target.ID,
	).Return(nil, nil).Once()

	// Последние три проверки завершились неудачно.
	history := []domain.Check{
		{TargetID: 1, IsSuccess: false},
		{TargetID: 1, IsSuccess: false},
		{TargetID: 1, IsSuccess: false},
	}

	checkRepo.On(
		"GetLastChecks",
		mock.Anything,
		target.ID,
		3,
	).Return(history, nil).Once()

	// Ожидаем создание нового OPEN-инцидента.
	incidentRepo.On(
		"CreateIncident",
		mock.Anything,
		mock.MatchedBy(func(incident *domain.Incident) bool {
			return incident != nil &&
				incident.TargetID == target.ID &&
				incident.Status == "OPEN" &&
				!incident.CreatedAt.IsZero()
		}),
	).Run(func(args mock.Arguments) {
		// Имитируем ID, который вернула бы PostgreSQL.
		incident := args.Get(1).(*domain.Incident)
		incident.ID = 42
	}).Return(nil).Once()

	// После создания инцидента ожидаем DOWN-алерт.
	alertCreator.On(
		"CreateAlert",
		mock.Anything,
		42,
		target.UserID,
		"DOWN",
		"🔴 Website unavailable",
	).Return(nil).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.NoError(t, err)

	checkRepo.AssertExpectations(t)
	incidentRepo.AssertExpectations(t)
	alertCreator.AssertExpectations(t)
}

func TestProcessCheck_AlreadyOpenIncident(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     1,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       200,
		Answer:       "Unexpected status",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    false,
	}

	// Для этого сайта уже существует открытый инцидент.
	existingIncident := &domain.Incident{
		ID:        42,
		TargetID:  target.ID,
		Status:    "OPEN",
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}

	// Новая неудачная проверка всё равно сохраняется.
	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.MatchedBy(func(check *domain.Check) bool {
			return check != nil &&
				check.TargetID == target.ID &&
				!check.IsSuccess
		}),
	).Return(nil).Once()

	// Репозиторий сообщает, что инцидент уже открыт.
	incidentRepo.On(
		"GetOpenIncident",
		mock.Anything,
		target.ID,
	).Return(existingIncident, nil).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.NoError(t, err)

	checkRepo.AssertExpectations(t)
	incidentRepo.AssertExpectations(t)

	// Повторно запрашивать историю не требуется.
	checkRepo.AssertNotCalled(
		t,
		"GetLastChecks",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)

	// Второй инцидент создаваться не должен.
	incidentRepo.AssertNumberOfCalls(
		t,
		"CreateIncident",
		0,
	)

	// Повторное уведомление DOWN не отправляем.
	alertCreator.AssertNumberOfCalls(
		t,
		"CreateAlert",
		0,
	)

	// Уже открытый инцидент не закрываем.
	incidentRepo.AssertNumberOfCalls(
		t,
		"CloseIncident",
		0,
	)
}

func TestProcessCheck_Recovery(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     6,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       200,
		Answer:       "OK",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    true,
	}

	// У сайта уже есть открытый инцидент.
	openIncident := &domain.Incident{
		ID:        42,
		TargetID:  target.ID,
		Status:    "OPEN",
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}

	// Успешную проверку сохраняем.
	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.MatchedBy(func(check *domain.Check) bool {
			return check != nil &&
				check.TargetID == target.ID &&
				check.IsSuccess &&
				check.Status == 200
		}),
	).Return(nil).Once()

	// Находим существующий открытый инцидент.
	incidentRepo.On(
		"GetOpenIncident",
		mock.Anything,
		target.ID,
	).Return(openIncident, nil).Once()

	// Ожидаем закрытие именно этого инцидента.
	incidentRepo.On(
		"CloseIncident",
		mock.Anything,
		openIncident.ID,
		mock.MatchedBy(func(closedAt time.Time) bool {
			return closedAt.After(openIncident.CreatedAt)
		}),
	).Return(nil).Once()

	// После закрытия создаётся UP-алерт.
	alertCreator.On(
		"CreateAlert",
		mock.Anything,
		openIncident.ID,
		target.UserID,
		"UP",
		"🟢 Website is available again",
	).Return(nil).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.NoError(t, err)

	checkRepo.AssertExpectations(t)
	incidentRepo.AssertExpectations(t)
	alertCreator.AssertExpectations(t)

	// Новый инцидент не создаём.
	incidentRepo.AssertNumberOfCalls(
		t,
		"CreateIncident",
		0,
	)

	// Историю ошибок при восстановлении не запрашиваем.
	checkRepo.AssertNumberOfCalls(
		t,
		"GetLastChecks",
		0,
	)
}

func TestProcessCheck_SaveCheckError(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     1,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       0,
		Answer:       "Connection refused",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    false,
	}

	// Имитируем ошибку PostgreSQL.
	expectedErr := errors.New("database unavailable")

	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.MatchedBy(func(check *domain.Check) bool {
			return check != nil &&
				check.TargetID == target.ID &&
				!check.IsSuccess
		}),
	).Return(expectedErr).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.ErrorIs(t, err, expectedErr)

	checkRepo.AssertExpectations(t)

	// После ошибки сохранения обработка должна остановиться.
	checkRepo.AssertNumberOfCalls(
		t,
		"GetLastChecks",
		0,
	)

	require.Empty(t, incidentRepo.Calls)
	require.Empty(t, alertCreator.Calls)
}

func TestProcessCheck_GetOpenIncidentError(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     1,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       0,
		Answer:       "Connection refused",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    false,
	}

	expectedErr := errors.New("failed to get open incident")

	// Результат проверки успешно сохраняется.
	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.MatchedBy(func(check *domain.Check) bool {
			return check != nil &&
				check.TargetID == target.ID &&
				!check.IsSuccess
		}),
	).Return(nil).Once()

	// Но получить открытый инцидент не удалось.
	incidentRepo.On(
		"GetOpenIncident",
		mock.Anything,
		target.ID,
	).Return(nil, expectedErr).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.ErrorIs(t, err, expectedErr)

	checkRepo.AssertExpectations(t)
	incidentRepo.AssertExpectations(t)

	// При ошибке дальше не идём.
	checkRepo.AssertNumberOfCalls(t, "GetLastChecks", 0)
	incidentRepo.AssertNumberOfCalls(t, "CreateIncident", 0)
	incidentRepo.AssertNumberOfCalls(t, "CloseIncident", 0)
	alertCreator.AssertNumberOfCalls(t, "CreateAlert", 0)
}

func TestProcessCheck_CreateAlertError(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     6,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       0,
		Answer:       "Connection refused",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    false,
	}

	expectedErr := errors.New("failed to create alert")

	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()

	incidentRepo.On(
		"GetOpenIncident",
		mock.Anything,
		target.ID,
	).Return(nil, nil).Once()

	history := []domain.Check{
		{TargetID: 6, IsSuccess: false},
		{TargetID: 6, IsSuccess: false},
		{TargetID: 6, IsSuccess: false},
	}

	checkRepo.On(
		"GetLastChecks",
		mock.Anything,
		target.ID,
		3,
	).Return(history, nil).Once()

	incidentRepo.On(
		"CreateIncident",
		mock.Anything,
		mock.MatchedBy(func(incident *domain.Incident) bool {
			return incident != nil &&
				incident.TargetID == target.ID &&
				incident.Status == "OPEN"
		}),
	).Run(func(args mock.Arguments) {
		incident := args.Get(1).(*domain.Incident)
		incident.ID = 42
	}).Return(nil).Once()

	// Имитируем ошибку создания алерта.
	alertCreator.On(
		"CreateAlert",
		mock.Anything,
		42,
		target.UserID,
		"DOWN",
		"🔴 Website unavailable",
	).Return(expectedErr).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.ErrorIs(t, err, expectedErr)

	checkRepo.AssertExpectations(t)
	incidentRepo.AssertExpectations(t)
	alertCreator.AssertExpectations(t)

	// Созданный инцидент не должен закрываться.
	incidentRepo.AssertNumberOfCalls(
		t,
		"CloseIncident",
		0,
	)
}

func TestProcessCheck_CloseIncidentError(t *testing.T) {
	// Arrange
	checkRepo := new(MockCheckRepository)
	incidentRepo := new(MockIncidentRepository)
	alertCreator := new(MockAlertCreator)

	uc := NewProcessCheckUseCase(
		checkRepo,
		incidentRepo,
		alertCreator,
	)

	target := &domain.Target{
		ID:     1,
		UserID: 1,
	}

	result := domain.CheckResult{
		Status:       200,
		Answer:       "OK",
		ResponseTime: 150 * time.Millisecond,
		CheckedAt:    time.Now(),
		IsSuccess:    true,
	}

	openIncident := &domain.Incident{
		ID:        42,
		TargetID:  target.ID,
		Status:    "OPEN",
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}

	expectedErr := errors.New("failed to close incident")

	// Результат успешной проверки сохранён.
	checkRepo.On(
		"SaveCheck",
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()

	// Находим открытый инцидент.
	incidentRepo.On(
		"GetOpenIncident",
		mock.Anything,
		target.ID,
	).Return(openIncident, nil).Once()

	// Но закрыть его не получается.
	incidentRepo.On(
		"CloseIncident",
		mock.Anything,
		openIncident.ID,
		mock.Anything,
	).Return(expectedErr).Once()

	// Act
	err := uc.ProcessCheck(
		context.Background(),
		target,
		result,
		3,
	)

	// Assert
	require.ErrorIs(t, err, expectedErr)

	checkRepo.AssertExpectations(t)
	incidentRepo.AssertExpectations(t)

	// Нельзя сообщать о восстановлении,
	// если закрытие инцидента завершилось ошибкой.
	require.Empty(t, alertCreator.Calls)

	incidentRepo.AssertNumberOfCalls(
		t,
		"CreateIncident",
		0,
	)
}
