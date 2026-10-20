package domain

import (
	"errors"
	"time"
)

type Incident struct {
	ID        int
	TargetID  int
	Status    string
	CreatedAt time.Time
	ClosedAt  time.Time
}

// Duration returns the elapsed duration for an open incident
// or the total downtime for a closed incident.
func (i *Incident) Duration() time.Duration {
	if i.Status == "OPEN" {
		return time.Since(i.CreatedAt)
	}
	return i.ClosedAt.Sub(i.CreatedAt)
}

func (i *Incident) Validate() error {
	if i.TargetID <= 0 {
		return errors.New("the incident must be linked to an existing TargetID")
	}
	if i.Status != "OPEN" && i.Status != "CLOSED" {
		return errors.New("incorrect incident status; it must be OPEN or CLOSED")
	}
	if i.Status == "CLOSED" && i.ClosedAt.Before(i.CreatedAt) {
		return errors.New("the time an incident is closed cannot be earlier than the time it was opened")
	}
	return nil
}
