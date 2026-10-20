package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	AlertTypeDown = "DOWN"
	AlertTypeUp   = "UP"
)

type Alert struct {
	ID         int
	IncidentID int
	Type       string
	Message    string
	CreatedAt  time.Time
}

func (a *Alert) Validate() error {
	if a.IncidentID <= 0 {
		return errors.New("the alert must be linked to an existing incident")
	}
	a.Type = strings.TrimSpace(a.Type)
	if a.Type != AlertTypeDown && a.Type != AlertTypeUp {
		return errors.New("the alert type must be DOWN or UP")
	}
	a.Message = strings.TrimSpace(a.Message)
	if a.Message == "" {
		return errors.New("the alert message cannot be empty")
	}

	return nil
}
