package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Uptime_Monitor/internal/domain"
)

// buildMonitorIncidentsText collects recent incidents
// and formats them for Telegram presentation.
func (b *Bot) buildMonitorIncidentsText(ctx context.Context, target *domain.Target) (string, error) {
	incidents, err := b.listIncidents.ListIncidents(
		ctx,
		target.ID,
		5,
	)
	if err != nil {
		return "", fmt.Errorf(
			"list incidents: %w",
			err,
		)
	}

	var response strings.Builder

	if len(incidents) == 0 {
		fmt.Fprintf(
			&response,
			"✅ No incidents recorded for monitor #%d.",
			target.ID,
		)

		return response.String(), nil
	}

	fmt.Fprintf(
		&response,
		"🚨 Incidents for monitor #%d\n"+
			"Address: %s\n\n",
		target.ID,
		target.Address,
	)

	for _, incident := range incidents {
		if incident.Status == "OPEN" {
			fmt.Fprintf(
				&response,
				"🔴 Incident #%d — OPEN\n"+
					"Started: %s\n"+
					"Duration: %s and counting\n\n",
				incident.ID,
				incident.CreatedAt.UTC().Format(
					"2006-01-02 15:04:05 UTC",
				),
				formatObservedDuration(
					time.Since(incident.CreatedAt),
				),
			)

			continue
		}

		fmt.Fprintf(
			&response,
			"⚫ Incident #%d — CLOSED\n"+
				"Started: %s\n"+
				"Resolved: %s\n"+
				"Duration: %s\n\n",
			incident.ID,
			incident.CreatedAt.UTC().Format(
				"2006-01-02 15:04:05 UTC",
			),
			incident.ClosedAt.UTC().Format(
				"2006-01-02 15:04:05 UTC",
			),
			formatObservedDuration(
				incident.Duration(),
			),
		)
	}

	return response.String(), nil
}
