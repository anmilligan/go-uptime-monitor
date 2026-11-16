package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	checksTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "uptime_monitor_checks_total",
			Help: "Total number of checks performed by workers.",
		},
	)

	monitorsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "uptime_monitor_monitors_total",
			Help: "Current number of monitors by status.",
		},
		[]string{"status"},
	)

	checkLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "uptime_monitor_check_latency_seconds",
			Help:    "Check latency in seconds by monitor.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"monitor_id"},
	)

	openIncidents = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "uptime_monitor_open_incidents",
			Help: "Current number of open incidents.",
		},
	)

	schedulerLag = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "uptime_monitor_scheduler_lag_seconds",
			Help:    "Delay between the scheduled time and enqueue time of a check.",
			Buckets: prometheus.DefBuckets,
		},
	)

	uptimePercentage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "uptime_monitor_uptime_percentage",
			Help: "Uptime percentage by monitor and period.",
		},
		[]string{"monitor_id", "period"},
	)
)

var monitorStatuses = []string{
	"up",
	"down",
	"paused",
	"unknown",
	"degraded",
}

func init() {
	prometheus.MustRegister(
		checksTotal,
		monitorsTotal,
		checkLatency,
		openIncidents,
		schedulerLag,
		uptimePercentage,
	)
}

// RecordCheck increments the total number of completed checks.
func RecordCheck() {
	checksTotal.Inc()
}

// UpdateMonitorStatusCounts updates the current number of monitors
// for each supported status.
func UpdateMonitorStatusCounts(counts map[string]int) {
	for _, status := range monitorStatuses {
		monitorsTotal.
			WithLabelValues(status).
			Set(float64(counts[status]))
	}
}

// RecordCheckLatency records the check duration for a monitor.
func RecordCheckLatency(monitorID int, duration time.Duration) {
	checkLatency.
		WithLabelValues(strconv.Itoa(monitorID)).
		Observe(duration.Seconds())
}

// UpdateOpenIncidentsCount sets the current number of open incidents.
func UpdateOpenIncidentsCount(count int) {
	openIncidents.Set(float64(count))
}

// RecordSchedulerLag records the delay between a scheduled check
// and the time it was enqueued for execution.
func RecordSchedulerLag(lag time.Duration) {
	if lag < 0 {
		lag = 0
	}

	schedulerLag.Observe(
		lag.Seconds(),
	)
}

// ResetUptimePercentages removes all currently stored uptime label series.
func ResetUptimePercentages() {
	uptimePercentage.Reset()
}

// UpdateUptimePercentage sets the uptime percentage
// for a monitor and observation period.
func UpdateUptimePercentage(monitorID int, period string, percentage float64) {
	uptimePercentage.
		WithLabelValues(
			strconv.Itoa(monitorID),
			period,
		).
		Set(percentage)
}
