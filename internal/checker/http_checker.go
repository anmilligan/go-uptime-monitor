package checker

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{},
	}
}
func (c *HTTPChecker) Check(ctx context.Context, target *domain.Target) domain.CheckResult {
	startedAt := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, target.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		checkCtx,
		http.MethodGet,
		target.Address,
		nil,
	)

	if err != nil {
		return domain.CheckResult{
			Answer:       err.Error(),
			ResponseTime: time.Since(startedAt),
			CheckedAt:    time.Now(),
			IsSuccess:    false,
		}
	}

	resp, err := c.client.Do(req)

	if err != nil {
		return domain.CheckResult{
			Answer:       err.Error(),
			ResponseTime: time.Since(startedAt),
			CheckedAt:    time.Now(),
			IsSuccess:    false,
		}
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.CheckResult{
			Status:       resp.StatusCode,
			Answer:       err.Error(),
			ResponseTime: time.Since(startedAt),
			CheckedAt:    time.Now(),
			IsSuccess:    false,
		}
	}

	answer := string(body)

	isSuccess := resp.StatusCode == target.ExpectedStatus

	if target.ExpectedBody != "" {
		isSuccess = isSuccess &&
			strings.Contains(answer, target.ExpectedBody)
	}

	return domain.CheckResult{
		Status:       resp.StatusCode,
		Answer:       answer,
		ResponseTime: time.Since(startedAt),
		CheckedAt:    time.Now(),
		IsSuccess:    isSuccess,
	}
}
