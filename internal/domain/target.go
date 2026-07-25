package domain

import (
	"errors"
	"time"
)

type Target struct {
	ID             int           // уникальный номер в бд
	UserID         int           // айди пользователя, который добавил сайт
	URL            string        // адрес сайта
	Interval       time.Duration // как часто проверять
	Timeout        time.Duration // сколько ждать ответа
	ExpectedStatus int           // ожидаемый HTTP-статус
	ExpectedBody   string        // ожидаемая строка в теле ответа
	CreatedAt      time.Time     // когда этот монитор был создан
}

func (t *Target) Validate() error {
	if t.URL == "" {
		return errors.New("URL не может быть пустым")
	}
	if t.Interval < 10*time.Second {
		return errors.New("интервал проверки не может быть меньше 10 секунд")
	}
	if t.Timeout <= 0 {
		return errors.New("таймаут должен быть больше нуля")
	}
	return nil
}
