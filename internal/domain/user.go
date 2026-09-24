package domain

import (
	"errors"
	"strings"
	"time"
)

type User struct {
	ID         int
	TelegramID int64
	Username   string
	CreatedAt  time.Time
}

func (u *User) Validate() error {
	u.Username = strings.TrimSpace(u.Username)

	if u.Username == "" {
		return errors.New("The name cannot be empty")
	}
	if len(u.Username) < 3 {
		return errors.New("The username must contain at least 3 characters")
	}
	return nil
}
