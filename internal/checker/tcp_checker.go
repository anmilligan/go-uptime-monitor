package checker

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"net"
	"time"
)

type TCPChecker struct{}

func NewTCPChecker() *TCPChecker {
	return &TCPChecker{}
}

func (c *TCPChecker) Check(
	ctx context.Context,
	target *domain.Target,
) domain.CheckResult {
	startedAt := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, target.Timeout)
	defer cancel()

	var dialer net.Dialer

	conn, err := dialer.DialContext(
		checkCtx,
		"tcp",
		target.Address,
	)

	responseTime := time.Since(startedAt)

	if err != nil {
		return domain.CheckResult{
			Status:       0,
			Answer:       err.Error(),
			ResponseTime: responseTime,
			CheckedAt:    time.Now(),
			IsSuccess:    false,
		}
	}

	defer func() {
		_ = conn.Close()
	}()

	return domain.CheckResult{
		Status:       0,
		Answer:       "TCP connection established",
		ResponseTime: responseTime,
		CheckedAt:    time.Now(),
		IsSuccess:    true,
	}
}
