package checker

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"fmt"
	"strings"
	"time"
)

// TargetChecker routes a target to the appropriate checker based on its kind.
type TargetChecker struct {
	httpChecker *HTTPChecker
	tcpChecker  *TCPChecker
}

func NewTargetChecker(httpChecker *HTTPChecker, tcpChecker *TCPChecker) *TargetChecker {
	return &TargetChecker{
		httpChecker: httpChecker,
		tcpChecker:  tcpChecker,
	}
}

// Check performs a target check using the checker that matches the target kind.
func (c *TargetChecker) Check(ctx context.Context, target *domain.Target) domain.CheckResult {
	if target == nil {
		return domain.CheckResult{
			Status:    0,
			Answer:    "target cannot be nil",
			CheckedAt: time.Now(),
			IsSuccess: false,
		}
	}

	switch strings.ToLower(strings.TrimSpace(target.Kind)) {
	case "http":
		return c.httpChecker.Check(ctx, target)

	case "tcp":
		return c.tcpChecker.Check(ctx, target)

	default:
		return domain.CheckResult{
			Status: 0,
			Answer: fmt.Sprintf(
				"unsupported target kind: %s",
				target.Kind,
			),
			CheckedAt: time.Now(),
			IsSuccess: false,
		}
	}
}
