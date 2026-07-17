package domain

import (
	"errors"
	"strings"
	"time"
)

type User struct {
	id         int       // id в нашей бд
	telegramID int       // цифровой id телеграмма
	username   string    // никнейм в системе
	createdAt  time.Time // время регистрации
}

func (u *User) Valodate() error {
	u.username = strings.TrimSpace(u.username)

	if u.username == "" {
		return errors.New("Имя не может быть пустым")
	}
	if len(u.username) < 3 {
		return errors.New("Имя пользователя должно содержать минимум 3 символа")
	}
	return nil
}
