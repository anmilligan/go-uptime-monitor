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
		return errors.New("The address cannot be empty")
	}
	if t.Kind != "http" && t.Kind != "tcp" {
		return errors.New("The connection type must be HTTP or TCP")
	}
	if t.Interval < 10*time.Second {
		return errors.New("The check interval cannot be less than 10 seconds")
	}
	if t.Timeout <= 0 {
		return errors.New("The timeout must be greater than zero")
	}
	if t.Kind == "http" {
		if t.ExpectedStatus < 100 || t.ExpectedStatus > 600 {
			return errors.New("Invalid expected HTTP status")
		}
	}
	return nil
}
