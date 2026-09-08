package checker

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"fmt"
	"strings"
	"time"
)

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

func (c *TargetChecker) Check(
	ctx context.Context,
	target *domain.Target,
) domain.CheckResult {
	switch strings.ToLower(strings.TrimSpace(target.Kind)) {
	case "http":
		return c.httpChecker.Check(ctx, target)

	case "tcp":
		return c.tcpChecker.Check(ctx, target)

	default:
		return domain.CheckResult{
			Status:    0,
			Answer:    fmt.Sprintf("unsupported target kind: %s", target.Kind),
			CheckedAt: time.Now(),
			IsSuccess: false,
		}
	}
}
