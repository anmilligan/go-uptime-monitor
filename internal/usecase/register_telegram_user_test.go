package usecase

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"errors"
	"testing"
)

type fakeUserRepository struct {
	savedUser *domain.User
	saveErr   error
	getUser   *domain.User
	getErr    error
}

func (f *fakeUserRepository) GetByID(
	ctx context.Context,
	id int,
) (*domain.User, error) {
	if f.getUser == nil && f.getErr == nil {
		return nil, errors.New("unexpected GetByID call")
	}

	return f.getUser, f.getErr
}

func (f *fakeUserRepository) Upsert(
	ctx context.Context,
	user *domain.User,
) error {
	if f.saveErr != nil {
		return f.saveErr
	}

	f.savedUser = user

	user.ID = 5

	return nil
}

func TestRegisterTelegramUser_Success(t *testing.T) {
	repo := &fakeUserRepository{}

	uc := NewRegisterTelegramUserUseCase(repo)

	user, err := uc.Register(
		context.Background(),
		987654321,
		"test_user",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.savedUser == nil {
		t.Fatal("repository was not called")
	}

	if repo.savedUser.TelegramID != 987654321 {
		t.Errorf(
			"unexpected Telegram ID: %d",
			repo.savedUser.TelegramID,
		)
	}

	if user.Username != "test_user" {
		t.Errorf(
			"unexpected username: %s",
			user.Username,
		)
	}

	if user.ID != 5 {
		t.Errorf(
			"expected user ID 5, got %d",
			user.ID,
		)
	}
}

func TestRegisterTelegramUser_WithoutUsername(t *testing.T) {
	// Arrange
	repo := &fakeUserRepository{}
	uc := NewRegisterTelegramUserUseCase(repo)

	// Act
	user, err := uc.Register(
		context.Background(),
		987654321,
		"",
	)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.savedUser == nil {
		t.Fatal("user was not saved")
	}

	expectedUsername := "tg_987654321"

	if user.Username != expectedUsername {
		t.Errorf(
			"expected username %q, got %q",
			expectedUsername,
			user.Username,
		)
	}
}

func TestRegisterTelegramUser_RepositoryError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database unavailable")

	repo := &fakeUserRepository{
		saveErr: expectedErr,
	}

	uc := NewRegisterTelegramUserUseCase(repo)

	// Act
	user, err := uc.Register(
		context.Background(),
		987654321,
		"test_user",
	)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Errorf(
			"expected error %v, got %v",
			expectedErr,
			err,
		)
	}

	if user != nil {
		t.Errorf(
			"expected nil user, got %+v",
			user,
		)
	}
}
