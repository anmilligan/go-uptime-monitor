package domain

import (
	"errors"
	"time"
)

type Target struct {
	id        int           // уникальный номер в бд
	userID    int           // айди пользователя, который добавил сайт
	url       string        // адрес сайта
	interval  time.Duration // как часто проверять
	timeout   time.Duration // сколько ждать ответа
	createdAt time.Time     // когда этот монитор был создан
}

func (t *Target) Validate() error {
	if t.url == "" {
		return errors.New("url не может быть пустым")
	}
	if t.interval < 10*time.Second {
		return errors.New("интервал проверки не может быть меньше 10 секунд")
	}
	if t.timeout <= 0 {
		return errors.New("таймаут должен быть больше нуля")
	}
	return nil
}
