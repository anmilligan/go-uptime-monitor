package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Uptime_Monitor/internal/domain"
)

// buildMonitorDetailsText collects the current monitor state
// and formats it for Telegram presentation.
func (b *Bot) buildMonitorDetailsText(ctx context.Context, target *domain.Target) (string, error) {
	check, err := b.lastCheck.GetLastCheck(
		ctx,
		target.ID,
	)
	if err != nil {
		return "", fmt.Errorf(
			"get last check: %w",
			err,
		)
	}

	incident, err := b.openIncident.GetOpenIncident(
		ctx,
		target.ID,
	)
	if err != nil {
		return "", fmt.Errorf(
			"get open incident: %w",
			err,
		)
	}

	uptime24h, err := b.uptime.CalculateUptime(
		ctx,
		target.ID,
		target.CreatedAt,
		24*time.Hour,
	)
	if err != nil {
		return "", fmt.Errorf(
			"calculate 24h uptime: %w",
			err,
		)
	}

	uptime7d, err := b.uptime.CalculateUptime(
		ctx,
		target.ID,
		target.CreatedAt,
		7*24*time.Hour,
	)
	if err != nil {
		return "", fmt.Errorf(
			"calculate 7d uptime: %w",
			err,
		)
	}

	var response strings.Builder

	fmt.Fprintf(
		&response,
		"🌐 Monitor #%d\n\n"+
			"Address: %s\n"+
			"Type: %s\n"+
			"Interval: %d seconds\n",
		target.ID,
		target.Address,
		strings.ToUpper(target.Kind),
		int64(target.Interval/time.Second),
	)

	if target.Active {
		response.WriteString(
			"Monitoring: ▶️ ACTIVE\n",
		)
	} else {
		response.WriteString(
			"Monitoring: ⏸ PAUSED\n",
		)
	}

	switch {
	case !target.Active:
		response.WriteString(
			"Status: ⏸ PAUSED\n",
		)

	case incident != nil:
		response.WriteString(
			"Status: 🔴 DOWN\n",
		)

	case check == nil:
		response.WriteString(
			"Status: ⚪ UNKNOWN\n",
		)

	case check.IsSuccess:
		response.WriteString(
			"Status: 🟢 UP\n",
		)

	default:
		response.WriteString(
			"Status: 🟠 Recent check failed; no open incident\n",
		)
	}

	if check != nil {
		if check.IsSuccess {
			response.WriteString(
				"Last check: 🟢 Successful\n",
			)
		} else {
			response.WriteString(
				"Last check: 🔴 Failed\n",
			)
		}

		if target.Kind == "http" &&
			check.Status != 0 {

			fmt.Fprintf(
				&response,
				"HTTP status: %d\n",
				check.Status,
			)
		}

		fmt.Fprintf(
			&response,
			"Latency: %d ms\n"+
				"Checked at: %s\n",
			check.ResponseTime.Milliseconds(),
			check.CheckedAt.UTC().Format(
				"2006-01-02 15:04:05 UTC",
			),
		)
	}

	response.WriteString("\nUptime:\n")

	if uptime24h.HasData {
		fmt.Fprintf(
			&response,
			"24h: %.2f%% (observed %s)\n",
			uptime24h.Percentage,
			formatObservedDuration(
				uptime24h.ObservedFor,
			),
		)
	} else {
		response.WriteString(
			"24h: not enough data yet\n",
		)
	}

	if uptime7d.HasData {
		fmt.Fprintf(
			&response,
			"7d: %.2f%% (observed %s)\n",
			uptime7d.Percentage,
			formatObservedDuration(
				uptime7d.ObservedFor,
			),
		)
	} else {
		response.WriteString(
			"7d: not enough data yet\n",
		)
	}

	if incident != nil {
		fmt.Fprintf(
			&response,
			"\nOpen incident: #%d\n"+
				"Started at: %s\n",
			incident.ID,
			incident.CreatedAt.UTC().Format(
				"2006-01-02 15:04:05 UTC",
			),
		)
	} else {
		response.WriteString(
			"\nOpen incident: None\n",
		)
	}

	return response.String(), nil
}
