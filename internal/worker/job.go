package worker

import "Uptime_Monitor/internal/domain"

// Job represents a target check task processed by the worker pool.
type Job struct {
	Target domain.Target
}
