package telegram

import (
	"fmt"
	"time"
)

// formatObservedDuration formats a duration into a compact
// human-readable representation using days, hours, and minutes.
func formatObservedDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0m"
	}

	totalMinutes := int64(duration / time.Minute)

	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60

	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd %dh", days, hours)

	case days > 0:
		return fmt.Sprintf("%dd", days)

	case hours > 0 && minutes > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)

	case hours > 0:
		return fmt.Sprintf("%dh", hours)

	default:
		return fmt.Sprintf("%dm", minutes)
	}
}
