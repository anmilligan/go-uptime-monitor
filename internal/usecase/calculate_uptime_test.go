package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeUptimeIncidentRepository struct {
	incidents []domain.Incident
	err       error
}

func (f *fakeUptimeIncidentRepository) GetIncidentsForPeriod(ctx context.Context, targetID int, from time.Time, to time.Time) ([]domain.Incident, error) {
	return f.incidents, f.err
}

func fixedUptimeNow() time.Time {
	return time.Date(
		2026, 10, 2,
		12, 0, 0, 0,
		time.UTC,
	)
}

func TestCalculateUptime_NoIncidents(t *testing.T) {
	// Arrange
	repo := &fakeUptimeIncidentRepository{}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	targetCreatedAt := fixedUptimeNow().Add(-48 * time.Hour)

	// Act
	result, err := uc.CalculateUptime(
		context.Background(),
		1,
		targetCreatedAt,
		24*time.Hour,
	)

	// Assert
	require.NoError(t, err)
	require.True(t, result.HasData)
	require.Equal(t, 24*time.Hour, result.ObservedFor)
	require.Equal(t, time.Duration(0), result.Downtime)
	require.Equal(t, 100.0, result.Percentage)
}

func TestCalculateUptime_ClosedIncident(t *testing.T) {
	// Arrange
	now := fixedUptimeNow()

	repo := &fakeUptimeIncidentRepository{
		incidents: []domain.Incident{
			{
				ID:        1,
				TargetID:  1,
				Status:    "CLOSED",
				CreatedAt: now.Add(-12 * time.Hour),
				ClosedAt:  now.Add(-11*time.Hour - 30*time.Minute),
			},
		},
	}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	targetCreatedAt := now.Add(-48 * time.Hour)

	// Act
	result, err := uc.CalculateUptime(
		context.Background(),
		1,
		targetCreatedAt,
		24*time.Hour,
	)

	// Assert
	require.NoError(t, err)

	require.Equal(
		t,
		30*time.Minute,
		result.Downtime,
	)

	require.Equal(
		t,
		24*time.Hour,
		result.ObservedFor,
	)

	require.InDelta(
		t,
		97.9167,
		result.Percentage,
		0.001,
	)
}

func TestCalculateUptime_OpenIncident(t *testing.T) {
	// Arrange
	now := fixedUptimeNow()

	repo := &fakeUptimeIncidentRepository{
		incidents: []domain.Incident{
			{
				ID:        1,
				TargetID:  1,
				Status:    "OPEN",
				CreatedAt: now.Add(-2 * time.Hour),
			},
		},
	}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	targetCreatedAt := now.Add(-48 * time.Hour)

	// Act
	result, err := uc.CalculateUptime(
		context.Background(),
		1,
		targetCreatedAt,
		24*time.Hour,
	)

	// Assert
	require.NoError(t, err)

	require.Equal(
		t,
		2*time.Hour,
		result.Downtime,
	)

	require.InDelta(
		t,
		91.6667,
		result.Percentage,
		0.001,
	)
}

func TestCalculateUptime_TargetCreatedInsidePeriod(t *testing.T) {
	// Arrange
	now := fixedUptimeNow()

	repo := &fakeUptimeIncidentRepository{}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	targetCreatedAt := now.Add(-6 * time.Hour)

	// Act
	result, err := uc.CalculateUptime(
		context.Background(),
		1,
		targetCreatedAt,
		24*time.Hour,
	)

	// Assert
	require.NoError(t, err)

	require.Equal(
		t,
		6*time.Hour,
		result.ObservedFor,
	)

	require.Equal(
		t,
		time.Duration(0),
		result.Downtime,
	)

	require.Equal(
		t,
		100.0,
		result.Percentage,
	)
}

func TestCalculateUptime_IncidentStartedBeforePeriod(t *testing.T) {
	// Arrange
	now := fixedUptimeNow()

	repo := &fakeUptimeIncidentRepository{
		incidents: []domain.Incident{
			{
				ID:       1,
				TargetID: 1,
				Status:   "CLOSED",

				// Начался за 26 часов до now.
				CreatedAt: now.Add(-26 * time.Hour),

				// Закончился за 23 часа до now.
				ClosedAt: now.Add(-23 * time.Hour),
			},
		},
	}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	targetCreatedAt := now.Add(-72 * time.Hour)

	// Act
	result, err := uc.CalculateUptime(
		context.Background(),
		1,
		targetCreatedAt,
		24*time.Hour,
	)

	// Assert
	require.NoError(t, err)

	// Сам инцидент длился 3 часа,
	// но в наши последние 24 часа попал только 1 час.
	require.Equal(
		t,
		1*time.Hour,
		result.Downtime,
	)

	require.InDelta(
		t,
		95.8333,
		result.Percentage,
		0.001,
	)
}

func TestCalculateUptime_InvalidTargetID(t *testing.T) {
	repo := &fakeUptimeIncidentRepository{}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	result, err := uc.CalculateUptime(
		context.Background(),
		0,
		fixedUptimeNow().Add(-24*time.Hour),
		24*time.Hour,
	)

	require.Error(t, err)
	require.Nil(t, result)
}

func TestCalculateUptime_InvalidPeriod(t *testing.T) {
	repo := &fakeUptimeIncidentRepository{}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	result, err := uc.CalculateUptime(
		context.Background(),
		1,
		fixedUptimeNow().Add(-24*time.Hour),
		0,
	)

	require.Error(t, err)
	require.Nil(t, result)
}

func TestCalculateUptime_RepositoryError(t *testing.T) {
	expectedErr := errors.New("database unavailable")

	repo := &fakeUptimeIncidentRepository{
		err: expectedErr,
	}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	result, err := uc.CalculateUptime(
		context.Background(),
		1,
		fixedUptimeNow().Add(-48*time.Hour),
		24*time.Hour,
	)

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, result)
}

func TestCalculateUptime_NoObservationTimeYet(t *testing.T) {
	now := fixedUptimeNow()

	repo := &fakeUptimeIncidentRepository{}

	uc := NewCalculateUptimeUseCase(repo)
	uc.now = fixedUptimeNow

	result, err := uc.CalculateUptime(
		context.Background(),
		1,
		now,
		24*time.Hour,
	)

	require.NoError(t, err)

	require.False(t, result.HasData)
	require.Equal(t, time.Duration(0), result.ObservedFor)
	require.Equal(t, time.Duration(0), result.Downtime)
	require.Equal(t, 0.0, result.Percentage)
}
