package domain

import (
	"errors"
	"time"
)

type Incident struct {
	ID        int
	TargetID  int       // ID сайта, который упал
	Status    string    // Статус: "OPEN" (лежит прямо сейчас) или "CLOSED" (работа восстановлена)
	CreatedAt time.Time // Время начала падения (когда зафиксировали N неудач подряд)
	ClosedAt  time.Time // Время, когда сайт снова ожил
}

func (i *Incident) Duration() time.Duration {
	if i.Status == "OPEN" {
		return time.Since(i.CreatedAt)
	}
	return i.ClosedAt.Sub(i.CreatedAt)
}

func (i *Incident) Validate() error {
	if i.TargetID <= 0 {
		return errors.New("The incident must be linked to an existing TargetID")
	}
	if i.Status != "OPEN" && i.Status != "CLOSED" {
		return errors.New("Incorrect incident status; it must be OPEN or CLOSED")
	}
	if i.Status == "CLOSED" && i.ClosedAt.Before(i.CreatedAt) {
		return errors.New("The time an incident is closed cannot be earlier than the time it was opened")
	}
	return nil
}
