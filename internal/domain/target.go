package domain

import (
	"errors"
	"time"
)

type Target struct {
	ID             int           // уникальный номер в бд
	UserID         int           // айди пользователя, который добавил сайт
	Kind           string        // тип проверки: http или tcp
	Address        string        // URL для HTTP или host:port для TCP
	Interval       time.Duration // как часто проверять
	Timeout        time.Duration // сколько ждать ответа
	ExpectedStatus int           // ожидаемый HTTP-статус
	ExpectedBody   string        // ожидаемая строка в теле ответа
	CreatedAt      time.Time     // когда этот монитор был создан
}

func (t *Target) Validate() error {
	if t.Address == "" {
		return errors.New("адрес не может быть пустым")
	}
	if t.Kind != "http" && t.Kind != "tcp" {
		return errors.New("тип проверки должен быть http или tcp")
	}
	if t.Interval < 10*time.Second {
		return errors.New("интервал проверки не может быть меньше 10 секунд")
	}
	if t.Timeout <= 0 {
		return errors.New("таймаут должен быть больше нуля")
	}
	if t.Kind == "http" {
		if t.ExpectedStatus < 100 || t.ExpectedStatus > 600 {
			return errors.New("некорректный ожидаемый HTTP-статус")
		}
	}
	return nil
}
