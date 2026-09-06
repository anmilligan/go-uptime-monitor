package domain

import (
	"errors"
	"strings"
	"time"
)

type Alert struct {
	ID         int
	IncidentID int       // ID проверяемого сайта, который упал
	Type       string    // Тип ошибки (например: Timeout)
	Message    string    // Читаемое описание
	CreatedAt  time.Time // Точное время ошибки
}

func (a *Alert) Validate() error {
	if a.IncidentID <= 0 {
		return errors.New("The alert must be linked to an existing TargetID")
	}
	a.Type = strings.TrimSpace(a.Type)
	if a.Type == "" {
		return errors.New("The Alert type cannot be empty")
	}
	if strings.TrimSpace(a.Message) == "" {
		return errors.New("The error description cannot be empty")
	}
	return nil
}
