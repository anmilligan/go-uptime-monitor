package domain

import "time"

type ScheduledTarget struct {
	Target      Target
	ScheduledAt time.Time
}
