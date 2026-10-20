package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"Uptime_Monitor/internal/usecase"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleAdd parses the /add command and creates an HTTP monitor.
func (b *Bot) handleAdd(ctx context.Context, message *tgbotapi.Message) error {
	args := strings.Fields(message.CommandArguments())

	if len(args) == 0 || len(args) > 2 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"Usage: /add https://example.com 60\n"+
				"Interval is optional (default: 60 seconds).",
		)
	}

	address := args[0]

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

		if target.Active {
			response.WriteString(
				"Monitoring: ▶️ ACTIVE\n",
			)
		} else {
			response.WriteString(
				"Monitoring: ⏸ PAUSED\n",
			)
		}

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
func (b *Bot) handleStatus(
	ctx context.Context,
	message *tgbotapi.Message,
) error {
	targetID, userMessage :=
		parseMonitorIDArgument(
			message,
			"status",
		)

	if userMessage != "" {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			userMessage,
		)
	}

	target, err := b.getOwnedTarget(
		ctx,
		message.From.ID,
		message.From.UserName,
		targetID,
	)
	if err != nil {
		return fmt.Errorf(
			"get owned target for /status: %w",
			err,
		)
	}

	if target == nil {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor not found or access denied.",
		)
	}

	response, err := b.buildMonitorDetailsText(
		ctx,
		target,
	)
	if err != nil {
		return fmt.Errorf(
			"build monitor details: %w",
			err,
		)
	}

	return b.notifier.Send(
		ctx,
		message.Chat.ID,
		response,
	)
}

// handleIncidents handles the /incidents command
// and sends recent incidents for a monitor owned by the user.
func (b *Bot) handleIncidents(ctx context.Context, message *tgbotapi.Message) error {
	targetID, userMessage :=
		parseMonitorIDArgument(
			message,
			"incidents",
		)

	if userMessage != "" {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			userMessage,
		)
	}

	target, err := b.getOwnedTarget(
		ctx,
		message.From.ID,
		message.From.UserName,
		targetID,
	)
	if err != nil {
		return fmt.Errorf(
			"get owned target for /incidents: %w",
			err,
		)
	}

	if target == nil {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor not found or access denied.",
		)
	}

	response, err := b.buildMonitorIncidentsText(
		ctx,
		target,
	)
	if err != nil {
		return fmt.Errorf(
			"build monitor incidents: %w",
			err,
		)
	}

	return b.notifier.Send(
		ctx,
		message.Chat.ID,
		response,
	)
}

// handleRemove handles the /remove command and deletes
// a monitor owned by the current user.
func (b *Bot) handleRemove(ctx context.Context, message *tgbotapi.Message) error {
	targetID, userMessage :=
		parseMonitorIDArgument(
			message,
			"remove",
		)

	if userMessage != "" {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			userMessage,
		)
	}

	target, err := b.getOwnedTarget(
		ctx,
		message.From.ID,
		message.From.UserName,
		targetID,
	)
	if err != nil {
		return fmt.Errorf(
			"get owned target for /remove: %w",
			err,
		)
	}

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
	targetID, userMessage :=
		parseMonitorIDArgument(
			message,
			"pause",
		)

	if userMessage != "" {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			userMessage,
		)
	}

	target, err := b.getOwnedTarget(
		ctx,
		message.From.ID,
		message.From.UserName,
		targetID,
	)
	if err != nil {
		return fmt.Errorf(
			"get owned target for /pause: %w",
			err,
		)
	}

	if target == nil {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"❌ Monitor not found or access denied.",
		)
	}

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
	targetID, userMessage :=
		parseMonitorIDArgument(
			message,
			"resume",
		)

	if userMessage != "" {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			userMessage,
		)
	}

	target, err := b.getOwnedTarget(
		ctx,
		message.From.ID,
		message.From.UserName,
		targetID,
	)
	if err != nil {
		return fmt.Errorf(
			"get owned target for /resume: %w",
			err,
		)
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
