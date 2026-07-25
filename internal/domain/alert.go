package domain

import (
	"errors"
	"strings"
	"time"
)

type Alert struct {
	ID        int
	TargetID  int       // ID проверяемого сайта, который упал
	Type      string    // Тип ошибки (например: Timeout)
	Message   string    // Читаемое описание
	CreatedAt time.Time // Точное время ошибки
}

func (a *Alert) Validate() error {
	if a.TargetID <= 0 {
		return errors.New("Алерт должен быть привязан к существующему TargetID")
	}
	a.Type = strings.TrimSpace(a.Type)
	if a.Type == "" {
		return errors.New("Тип Алерт не может быть пустым")
	}
	if strings.TrimSpace(a.Message) == "" {
		return errors.New("Описание ошибки не может быть пустым")
	}
	return nil
}
