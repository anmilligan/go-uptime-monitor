package metrics

import (
	"Uptime_Monitor/internal/usecase"
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// IMonitorStatusRepository defines the metrics data queries
// required by the Prometheus HTTP handler.
type IMonitorStatusRepository interface {
	GetMonitorStatusCounts(ctx context.Context) (map[string]int, error)
	GetOpenIncidentsCount(ctx context.Context) (int, error)
	GetMonitorCreatedAt(ctx context.Context) (map[int]time.Time, error)
}

// ICalculateUptimeUseCase defines uptime calculation
// required for Prometheus uptime metrics.
type ICalculateUptimeUseCase interface {
	CalculateUptime(ctx context.Context, targetID int, targetCreatedAt time.Time, period time.Duration) (*usecase.UptimeResult, error)
}

// Handler creates an HTTP handler that collects current application metrics
// and exposes them in Prometheus format.
func Handler(repo IMonitorStatusRepository, uptime ICalculateUptimeUseCase) http.Handler {
	prometheusHandler := promhttp.Handler()

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			counts, err := repo.GetMonitorStatusCounts(r.Context())
			if err != nil {
				http.Error(w, "failed to collect monitor metrics", http.StatusServiceUnavailable)
				return
			}

			UpdateMonitorStatusCounts(counts)

			openIncidentsCount, err := repo.GetOpenIncidentsCount(r.Context())
			if err != nil {
				http.Error(w, "failed to collect incident metrics", http.StatusServiceUnavailable)
				return
			}

			UpdateOpenIncidentsCount(openIncidentsCount)

			monitors, err := repo.GetMonitorCreatedAt(
				r.Context(),
			)
			if err != nil {
				http.Error(
					w,
					"failed to collect uptime metrics",
					http.StatusServiceUnavailable,
				)
				return
			}

			ResetUptimePercentages()

			for monitorID, createdAt := range monitors {
				uptime24h, err := uptime.CalculateUptime(
					r.Context(),
					monitorID,
					createdAt,
					24*time.Hour,
				)
				if err != nil {
					http.Error(
						w,
						"failed to calculate 24h uptime",
						http.StatusServiceUnavailable,
					)
					return
				}

				if uptime24h.HasData {
					UpdateUptimePercentage(
						monitorID,
						"24h",
						uptime24h.Percentage,
					)
				}

				uptime7d, err := uptime.CalculateUptime(
					r.Context(),
					monitorID,
					createdAt,
					7*24*time.Hour,
				)
				if err != nil {
					http.Error(
						w,
						"failed to calculate 7d uptime",
						http.StatusServiceUnavailable,
					)
					return
				}

				if uptime7d.HasData {
					UpdateUptimePercentage(
						monitorID,
						"7d",
						uptime7d.Percentage,
					)
				}
			}

			prometheusHandler.ServeHTTP(w, r)
		},
	)
}
