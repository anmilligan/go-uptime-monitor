package domain

import (
	"errors"
	"time"
)

type Check struct {
	ID           int
	TargetID     int           // ID цели, которую проверяли
	Status       int           // Полученный HTTP-статус
	Answer       string        // Текст ошибки
	ResponseTime time.Duration // Время ответа сайта
	CheckedAt    time.Time     // Время проведения проверки
	IsSuccess    bool
}

func (c *Check) Validate() error {
	if c.TargetID <= 0 {
		return errors.New("Проверка должна осуществляться с существующим TargetID")
	}
	if c.Status != 0 && (c.Status < 100 || c.Status > 599) {
		return errors.New("Некорректный HTTP-статус ответа")
	}
	if c.ResponseTime < 0 {
		return errors.New("Время ответа не может быть отрицательным")
	}
	return nil
}
