package telegram

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/usecase"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleAdd parses the /add command and creates an HTTP monitor.
func (b *Bot) handleAdd(ctx context.Context, message *tgbotapi.Message) error {
	args := strings.Fields(message.CommandArguments())

	// Validate the command argument count.
	if len(args) == 0 || len(args) > 2 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"Usage: /add https://example.com 60\n"+
				"Interval is optional (default: 60 seconds).",
		)
	}

	address := args[0]

	// Validate the HTTP(S) address.
	parsedURL, err := url.ParseRequestURI(address)
	if err != nil ||
		parsedURL == nil ||
		(parsedURL.Scheme != "http" &&
			parsedURL.Scheme != "https") ||
		parsedURL.Hostname() == "" {

		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Please provide a valid HTTP or HTTPS URL.",
		)
	}

	// Use the default interval when it is not provided.
	intervalSeconds := 60

	if len(args) == 2 {
		intervalSeconds, err = strconv.Atoi(args[1])

		if err != nil ||
			intervalSeconds < 10 ||
			intervalSeconds > 86400 {

			return b.notifier.Send(
				ctx,
				message.Chat.ID,
				"❌ Interval must be between "+
					"10 and 86400 seconds.",
			)
		}
	}

	// Register the Telegram user and obtain the internal user ID.
	user, err := b.register.Register(
		ctx,
		message.From.ID,
		message.From.UserName,
	)
	if err != nil {
		return fmt.Errorf(
			"register telegram user: %w",
			err,
		)
	}

	// Register the Telegram user and obtain the internal user ID.
	// Create the monitor through the application use case.
	target, err := b.addTarget.AddTarget(
		ctx,
		usecase.AddTargetRequest{
			UserID:         user.ID,
			Kind:           "http",
			Address:        address,
			Interval:       time.Duration(intervalSeconds) * time.Second,
			Timeout:        5 * time.Second,
			ExpectedStatus: 200,
		},
	)

	if err != nil {
		slog.ErrorContext(
			ctx,
			"telegram failed to add monitor",
			"error", err,
		)

		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Could not create monitor. Please try again.",
		)
	}

	response := fmt.Sprintf(
		"✅ Monitor created!\n\n"+
			"🌐 %s\n"+
			"⏱ Interval: %d seconds\n"+
			"🔎 Type: HTTP\n"+
			"ID: %d",
		target.Address,
		intervalSeconds,
		target.ID,
	)

	return b.notifier.Send(
		ctx,
		message.Chat.ID,
		response,
	)
}

// handleList handles the /list command for the current Telegram user.
func (b *Bot) handleList(ctx context.Context, message *tgbotapi.Message) error {
	return b.handleListForUser(
		ctx,
		message.Chat.ID,
		message.From.ID,
		message.From.UserName,
	)
}

// handleListForUser builds and sends a summary of all monitors
// owned by the specified Telegram user.
func (b *Bot) handleListForUser(ctx context.Context, chatID int64, telegramUserID int64, username string) error {
	user, err := b.register.Register(
		ctx,
		telegramUserID,
		username,
	)
	if err != nil {
		return fmt.Errorf(
			"register user for /list: %w",
			err,
		)
	}

	targets, err := b.listTargets.ListTargets(
		ctx,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"list user monitors: %w",
			err,
		)
	}

	if len(targets) == 0 {
		return b.notifier.Send(
			ctx,
			chatID,
			"📋 You don't have any monitors yet.\n\n"+
				"Use /add https://example.com to create one.",
		)
	}

	var response strings.Builder

	fmt.Fprintf(
		&response,
		"📋 Your monitors (%d)\n\n",
		len(targets),
	)

	for _, target := range targets {
		fmt.Fprintf(
			&response,
			"🔹 #%d — %s\n"+
				"Type: %s\n"+
				"Interval: %d seconds\n",
			target.ID,
			target.Address,
			strings.ToUpper(target.Kind),
			int64(target.Interval/time.Second),
		)

		// Show whether the monitor participates in scheduled checks.
		if target.Active {
			response.WriteString(
				"Monitoring: ▶️ ACTIVE\n",
			)
		} else {
			response.WriteString(
				"Monitoring: ⏸ PAUSED\n",
			)
		}

		// Retrieve the most recent check for the current monitor.
		check, err := b.lastCheck.GetLastCheck(
			ctx,
			target.ID,
		)
		if err != nil {
			return fmt.Errorf(
				"get last check for target %d: %w",
				target.ID,
				err,
			)
		}

		if check == nil {
			response.WriteString(
				"Last check: ⚪ UNKNOWN\n\n",
			)
			continue
		}

		if check.IsSuccess {
			response.WriteString(
				"Last check: 🟢 Successful\n",
			)
		} else {
			response.WriteString(
				"Last check: 🔴 Failed\n",
			)
		}

		// Show the HTTP status only for HTTP monitors.
		if target.Kind == "http" && check.Status != 0 {
			fmt.Fprintf(
				&response,
				"HTTP status: %d\n",
				check.Status,
			)
		}

		fmt.Fprintf(
			&response,
			"Latency: %d ms\n"+
				"Checked at: %s\n\n",
			check.ResponseTime.Milliseconds(),
			check.CheckedAt.UTC().Format(
				"2006-01-02 15:04:05 UTC",
			),
		)
	}

	return b.notifier.Send(
		ctx,
		chatID,
		response.String(),
	)
}

// handleStatus handles the /status command and sends
// the current state of a monitor owned by the user.
func (b *Bot) handleStatus(ctx context.Context, message *tgbotapi.Message) error {
	args := strings.Fields(message.CommandArguments())

	if len(args) != 1 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"Usage: /status <monitor_id>",
		)
	}

	targetID, err := strconv.Atoi(args[0])
	if err != nil || targetID <= 0 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor ID must be a positive number.",
		)
	}

	// Retrieve the user’s internal ID.
	user, err := b.register.Register(
		ctx,
		message.From.ID,
		message.From.UserName,
	)
	if err != nil {
		return fmt.Errorf(
			"register user for /status: %w",
			err,
		)
	}

	// Search for a monitor only amongst those owned by the user.
	targets, err := b.listTargets.ListTargets(
		ctx,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"list user monitors: %w",
			err,
		)
	}

	var target *domain.Target

	for i := range targets {
		if targets[i].ID == targetID {
			target = &targets[i]
			break
		}
	}

	if target == nil {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor not found or access denied.",
		)
	}

	// We get the latest check.
	check, err := b.lastCheck.GetLastCheck(
		ctx,
		target.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"get last check: %w",
			err,
		)
	}

	// Retrieve any open incidents, if there are any.
	incident, err := b.openIncident.GetOpenIncident(
		ctx,
		target.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"get open incident: %w",
			err,
		)
	}

	// Calculate the uptime for the last 24 hours.
	uptime24h, err := b.uptime.CalculateUptime(
		ctx,
		target.ID,
		target.CreatedAt,
		24*time.Hour,
	)
	if err != nil {
		return fmt.Errorf(
			"calculate 24h uptime: %w",
			err,
		)
	}

	// Calculate the uptime for the last 7 days.
	uptime7d, err := b.uptime.CalculateUptime(
		ctx,
		target.ID,
		target.CreatedAt,
		7*24*time.Hour,
	)
	if err != nil {
		return fmt.Errorf(
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
			"Interval: %d seconds\n\n",
		target.ID,
		target.Address,
		strings.ToUpper(target.Kind),
		int64(target.Interval/time.Second),
	)

	// Determine the monitor’s current state.
	// PAUSED takes precedence because the monitor is currently
	// not participating in scheduled checks.
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
	// Information about the last check.
	if check != nil {
		if check.IsSuccess {
			response.WriteString(
				"Last check: Successful\n",
			)
		} else {
			response.WriteString(
				"Last check: Failed\n",
			)
		}

		if target.Kind == "http" && check.Status != 0 {
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

	// Uptime over 24 hours and 7 days.
	response.WriteString("\nUptime:\n")

	if uptime24h.HasData {
		fmt.Fprintf(
			&response,
			"24h: %.2f%% (observed %s)\n",
			uptime24h.Percentage,
			formatObservedDuration(uptime24h.ObservedFor),
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
			formatObservedDuration(uptime7d.ObservedFor),
		)
	} else {
		response.WriteString(
			"7d: not enough data yet\n",
		)
	}
	// Information about an open incident.
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

	return b.notifier.Send(
		ctx,
		message.Chat.ID,
		response.String(),
	)
}

// handleIncidents handles the /incidents command
// and sends recent incident history for a monitor owned by the user.
func (b *Bot) handleIncidents(ctx context.Context, message *tgbotapi.Message) error {
	args := strings.Fields(message.CommandArguments())

	if len(args) != 1 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"Usage: /incidents <monitor_id>",
		)
	}

	targetID, err := strconv.Atoi(args[0])
	if err != nil || targetID <= 0 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor ID must be a positive number.",
		)
	}

	user, err := b.register.Register(
		ctx,
		message.From.ID,
		message.From.UserName,
	)
	if err != nil {
		return fmt.Errorf(
			"register user for /incidents: %w",
			err,
		)
	}

	targets, err := b.listTargets.ListTargets(
		ctx,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"list user monitors: %w",
			err,
		)
	}

	var target *domain.Target

	for i := range targets {
		if targets[i].ID == targetID {
			target = &targets[i]
			break
		}
	}

	if target == nil {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor not found or access denied.",
		)
	}

	incidents, err := b.listIncidents.ListIncidents(
		ctx,
		target.ID,
		5,
	)
	if err != nil {
		return fmt.Errorf(
			"list incidents: %w",
			err,
		)
	}

	if len(incidents) == 0 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			fmt.Sprintf(
				"✅ No incidents recorded for monitor #%d.",
				target.ID,
			),
		)
	}

	var response strings.Builder

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

	return b.notifier.Send(
		ctx,
		message.Chat.ID,
		response.String(),
	)
}

// handleRemove handles the /remove command and deletes
// a monitor owned by the current user.
func (b *Bot) handleRemove(ctx context.Context, message *tgbotapi.Message) error {
	args := strings.Fields(message.CommandArguments())

	if len(args) != 1 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"Usage: /remove <monitor_id>",
		)
	}

	targetID, err := strconv.Atoi(args[0])
	if err != nil || targetID <= 0 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor ID must be a positive number.",
		)
	}

	user, err := b.register.Register(
		ctx,
		message.From.ID,
		message.From.UserName,
	)
	if err != nil {
		return fmt.Errorf(
			"register user for /remove: %w",
			err,
		)
	}

	// Retrieve only the monitors for the current user.
	targets, err := b.listTargets.ListTargets(
		ctx,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"list user monitors: %w",
			err,
		)
	}

	var target *domain.Target

	for i := range targets {
		if targets[i].ID == targetID {
			target = &targets[i]
			break
		}
	}

	// We do not allow the deletion of someone else’s monitor or a non-existent monitor.
	if target == nil {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor not found or access denied.",
		)
	}

	if err := b.deleteTarget.DeleteTarget(
		ctx,
		target.ID,
	); err != nil {
		return fmt.Errorf(
			"delete monitor: %w",
			err,
		)
	}

	return b.notifier.Send(
		ctx,
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Monitor #%d removed.\n%s",
			target.ID,
			target.Address,
		),
	)
}

// handlePause handles the /pause command and disables
// scheduled checks for a monitor owned by the current user.
func (b *Bot) handlePause(ctx context.Context, message *tgbotapi.Message) error {
	args := strings.Fields(message.CommandArguments())

	if len(args) != 1 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"Usage: /pause <monitor_id>",
		)
	}

	targetID, err := strconv.Atoi(args[0])
	if err != nil || targetID <= 0 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor ID must be a positive number.",
		)
	}

	// Retrieve the user’s internal ID.
	user, err := b.register.Register(
		ctx,
		message.From.ID,
		message.From.UserName,
	)
	if err != nil {
		return fmt.Errorf(
			"register user for /pause: %w",
			err,
		)
	}

	// Retrieve only the current user’s monitors.
	targets, err := b.listTargets.ListTargets(
		ctx,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"list user monitors: %w",
			err,
		)
	}

	var target *domain.Target

	for i := range targets {
		if targets[i].ID == targetID {
			target = &targets[i]
			break
		}
	}

	if target == nil {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor not found or access denied.",
		)
	}

	// Do not execute an unnecessary UPDATE if the monitor is already paused.
	if !target.Active {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			fmt.Sprintf(
				"⏸ Monitor #%d is already paused.",
				target.ID,
			),
		)
	}

	if err := b.pauseTarget.PauseTarget(
		ctx,
		target.ID,
	); err != nil {
		return fmt.Errorf(
			"pause monitor: %w",
			err,
		)
	}

	return b.notifier.Send(
		ctx,
		message.Chat.ID,
		fmt.Sprintf(
			"⏸ Monitor #%d paused.\n%s",
			target.ID,
			target.Address,
		),
	)
}

// handleResume handles the /resume command and re-enables
// scheduled checks for a monitor owned by the current user.
func (b *Bot) handleResume(ctx context.Context, message *tgbotapi.Message) error {
	args := strings.Fields(message.CommandArguments())

	if len(args) != 1 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"Usage: /resume <monitor_id>",
		)
	}

	targetID, err := strconv.Atoi(args[0])
	if err != nil || targetID <= 0 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor ID must be a positive number.",
		)
	}

	// Retrieve the user’s internal ID.
	user, err := b.register.Register(
		ctx,
		message.From.ID,
		message.From.UserName,
	)
	if err != nil {
		return fmt.Errorf(
			"register user for /resume: %w",
			err,
		)
	}

	// A paused monitor is still returned via ListByUserID(),
	// so the user can reactivate it.
	targets, err := b.listTargets.ListTargets(
		ctx,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"list user monitors: %w",
			err,
		)
	}

	var target *domain.Target

	for i := range targets {
		if targets[i].ID == targetID {
			target = &targets[i]
			break
		}
	}

	if target == nil {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor not found or access denied.",
		)
	}

	if target.Active {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			fmt.Sprintf(
				"▶️ Monitor #%d is already active.",
				target.ID,
			),
		)
	}

	if err := b.resumeTarget.ResumeTarget(
		ctx,
		target.ID,
	); err != nil {
		return fmt.Errorf(
			"resume monitor: %w",
			err,
		)
	}

	return b.notifier.Send(
		ctx,
		message.Chat.ID,
		fmt.Sprintf(
			"▶️ Monitor #%d resumed.\n%s",
			target.ID,
			target.Address,
		),
	)
}
