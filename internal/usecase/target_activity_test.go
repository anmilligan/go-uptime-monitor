package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockTargetActivityRepository struct {
	mock.Mock
}

func (m *MockTargetActivityRepository) Pause(
	ctx context.Context,
	id int,
) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTargetActivityRepository) Resume(
	ctx context.Context,
	id int,
) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestPauseTarget_Success(t *testing.T) {
	repo := new(MockTargetActivityRepository)
	uc := NewPauseTargetUseCase(repo)

	repo.On(
		"Pause",
		mock.Anything,
		7,
	).Return(nil).Once()

	err := uc.PauseTarget(
		context.Background(),
		7,
	)

	require.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestPauseTarget_InvalidID(t *testing.T) {
	repo := new(MockTargetActivityRepository)
	uc := NewPauseTargetUseCase(repo)

	err := uc.PauseTarget(
		context.Background(),
		0,
	)

	require.Error(t, err)

	repo.AssertNotCalled(
		t,
		"Pause",
		mock.Anything,
		mock.Anything,
	)
}

func TestResumeTarget_Success(t *testing.T) {
	repo := new(MockTargetActivityRepository)
	uc := NewResumeTargetUseCase(repo)

	repo.On(
		"Resume",
		mock.Anything,
		7,
	).Return(nil).Once()

	err := uc.ResumeTarget(
		context.Background(),
		7,
	)

	require.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestResumeTarget_RepositoryError(t *testing.T) {
	repo := new(MockTargetActivityRepository)
	uc := NewResumeTargetUseCase(repo)

	expectedErr := errors.New("database error")

	repo.On(
		"Resume",
		mock.Anything,
		7,
	).Return(expectedErr).Once()

	err := uc.ResumeTarget(
		context.Background(),
		7,
	)

	require.ErrorIs(t, err, expectedErr)

	repo.AssertExpectations(t)
}
