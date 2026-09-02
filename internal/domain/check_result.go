package domain

import "time"

type CheckResult struct {
	Status       int
	Answer       string
	ResponseTime time.Duration
	CheckedAt    time.Time
	IsSuccess    bool
}
