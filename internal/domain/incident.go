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
		return errors.New("Инцидент должен быть привязан к существующему TargetID")
	}
	if i.Status != "OPEN" && i.Status != "CLOSED" {
		return errors.New("Некорректный статус инцидента, должно быть OPEN или CLOSED")
	}
	if i.Status == "CLOSED" && i.ClosedAt.Before(i.CreatedAt) {
		return errors.New("Время закрытия инцидента не может быть раньше времени его открытия")
	}
	return nil
}
