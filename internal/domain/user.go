package domain

import (
	"errors"
	"strings"
	"time"
)

type User struct {
	ID         int       // id в нашей бд
	TelegramID int       // цифровой id телеграмма
	Username   string    // никнейм в системе
	CreatedAt  time.Time // время регистрации
}

func (u *User) Validate() error {
	u.Username = strings.TrimSpace(u.Username)

	if u.Username == "" {
		return errors.New("Имя не может быть пустым")
	}
	if len(u.Username) < 3 {
		return errors.New("Имя пользователя должно содержать минимум 3 символа")
	}
	return nil
}
