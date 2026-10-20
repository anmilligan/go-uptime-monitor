package domain

import (
	"errors"
	"time"
)

type Check struct {
	ID           int
	TargetID     int
	Status       int
	Answer       string // Текст ошибки
	ResponseTime time.Duration
	CheckedAt    time.Time
	IsSuccess    bool
}

// Validate checks whether the recorded check satisfies domain invariants.
func (c *Check) Validate() error {
	if c.TargetID <= 0 {
		return errors.New("the verification must be performed using the existing TargetID")
	}
	if c.Status != 0 && (c.Status < 100 || c.Status > 599) {
		return errors.New("Invalid HTTP Response Status")
	}
	if c.ResponseTime < 0 {
		return errors.New("response time cannot be negative")
	}
	return nil
}
