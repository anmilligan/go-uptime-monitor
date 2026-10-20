package telegram

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// sendMainMenu sends the main Telegram menu with inline actions.
func (b *Bot) sendMainMenu(ctx context.Context, chatID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"➕ Add monitor",
				"menu:add",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"📋 My monitors",
				"menu:list",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"❓ Help",
				"menu:help",
			),
		),
	)

	msg := tgbotapi.NewMessage(
		chatID,
		"🌐 Uptime Monitor\n\nWhat do you want to do?",
	)

	msg.ReplyMarkup = keyboard

	_, err := b.api.Send(msg)
	return err
}

// sendHelp sends the list of supported bot commands.
func (b *Bot) sendHelp(ctx context.Context, chatID int64) error {
	return b.notifier.Send(
		ctx,
		chatID,
		"📖 Uptime Monitor — Help\n\n"+
			"/start — Register and start using the bot\n"+
			"/help — Show this help message\n"+
			"/add <url> [interval] — Add an HTTP monitor\n"+
			"/list — Show your monitors\n"+
			"/status <id> — Show monitor details\n"+
			"/incidents <id> — Show recent incidents\n"+
			"/remove <id> — Remove a monitor\n"+
			"/pause <id> — Pause monitor checks\n"+
			"/resume <id> — Resume monitor checks",
	)
}

// sendMonitorSelection sends an inline menu containing
// the monitors owned by the current Telegram user.
func (b *Bot) sendMonitorSelection(ctx context.Context, chatID int64, telegramUserID int64, username string) error {
	user, err := b.register.Register(
		ctx,
		telegramUserID,
		username,
	)
	if err != nil {
		return fmt.Errorf("register user for monitor menu: %w", err)
	}

	targets, err := b.listTargets.ListTargets(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("list user monitors: %w", err)
	}

	if len(targets) == 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("‹ Back", "menu:main"),
			),
		)

		msg := tgbotapi.NewMessage(
			chatID,
			"📋 You don't have any monitors yet.",
		)
		msg.ReplyMarkup = keyboard

		_, err := b.api.Send(msg)
		return err
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(targets)+1)

	for _, target := range targets {
		rows = append(
			rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					monitorButtonLabel(target),
					fmt.Sprintf("monitor:%d", target.ID),
				),
			),
		)
	}

	rows = append(
		rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("‹ Back", "menu:main"),
		),
	)

	msg := tgbotapi.NewMessage(
		chatID,
		fmt.Sprintf("📋 Your monitors (%d)\n\nChoose a monitor:", len(targets)),
	)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err = b.api.Send(msg)
	return err
}

// monitorButtonLabel builds a compact label for a monitor selection button.
func monitorButtonLabel(target domain.Target) string {
	state := "▶️"
	if !target.Active {
		state = "⏸"
	}

	name := target.Address
	if parsed, err := url.Parse(target.Address); err == nil && parsed.Hostname() != "" {
		name = parsed.Hostname()
	}

	if len(name) > 30 {
		name = name[:27] + "..."
	}

	return fmt.Sprintf("%s #%d — %s", state, target.ID, name)
}

// sendMonitorDetails collects the current monitor state
// and sends it together with available actions.
func (b *Bot) sendMonitorDetails(ctx context.Context, chatID int64, target *domain.Target) error {
	check, err := b.lastCheck.GetLastCheck(ctx, target.ID)
	if err != nil {
		return fmt.Errorf("get last check: %w", err)
	}

	incident, err := b.openIncident.GetOpenIncident(ctx, target.ID)
	if err != nil {
		return fmt.Errorf("get open incident: %w", err)
	}

	uptime24h, err := b.uptime.CalculateUptime(
		ctx,
		target.ID,
		target.CreatedAt,
		24*time.Hour,
	)
	if err != nil {
		return fmt.Errorf("calculate 24h uptime: %w", err)
	}

	uptime7d, err := b.uptime.CalculateUptime(
		ctx,
		target.ID,
		target.CreatedAt,
		7*24*time.Hour,
	)
	if err != nil {
		return fmt.Errorf("calculate 7d uptime: %w", err)
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
		response.WriteString("Monitoring: ▶️ ACTIVE\n")
	} else {
		response.WriteString("Monitoring: ⏸ PAUSED\n")
	}

	switch {
	case !target.Active:
		response.WriteString("Status: ⏸ PAUSED\n")
	case incident != nil:
		response.WriteString("Status: 🔴 DOWN\n")
	case check == nil:
		response.WriteString("Status: ⚪ UNKNOWN\n")
	case check.IsSuccess:
		response.WriteString("Status: 🟢 UP\n")
	default:
		response.WriteString("Status: 🟠 Recent check failed; no open incident\n")
	}

	if check != nil {
		if check.IsSuccess {
			response.WriteString("Last check: 🟢 Successful\n")
		} else {
			response.WriteString("Last check: 🔴 Failed\n")
		}

		if target.Kind == "http" && check.Status != 0 {
			fmt.Fprintf(&response, "HTTP status: %d\n", check.Status)
		}

		fmt.Fprintf(
			&response,
			"Latency: %d ms\nChecked at: %s\n",
			check.ResponseTime.Milliseconds(),
			check.CheckedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		)
	}

	response.WriteString("\nUptime:\n")

	if uptime24h.HasData {
		fmt.Fprintf(
			&response,
			"24h: %.2f%% (observed %s)\n",
			uptime24h.Percentage,
			formatObservedDuration(uptime24h.ObservedFor),
		)
	} else {
		response.WriteString("24h: not enough data yet\n")
	}

	if uptime7d.HasData {
		fmt.Fprintf(
			&response,
			"7d: %.2f%% (observed %s)\n",
			uptime7d.Percentage,
			formatObservedDuration(uptime7d.ObservedFor),
		)
	} else {
		response.WriteString("7d: not enough data yet\n")
	}

	if incident != nil {
		fmt.Fprintf(
			&response,
			"\nOpen incident: #%d\nStarted at: %s\n",
			incident.ID,
			incident.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		)
	} else {
		response.WriteString("\nOpen incident: None\n")
	}

	actionButton := tgbotapi.NewInlineKeyboardButtonData(
		"⏸ Pause",
		fmt.Sprintf("pause:%d", target.ID),
	)
	if !target.Active {
		actionButton = tgbotapi.NewInlineKeyboardButtonData(
			"▶️ Resume",
			fmt.Sprintf("resume:%d", target.ID),
		)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🚨 Incidents",
				fmt.Sprintf("incidents:%d", target.ID),
			),
			actionButton,
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🗑 Remove",
				fmt.Sprintf("remove:%d", target.ID),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("‹ Back to monitors", "menu:list"),
			tgbotapi.NewInlineKeyboardButtonData("⌂ Main menu", "menu:main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, response.String())
	msg.ReplyMarkup = keyboard

	_, err = b.api.Send(msg)
	return err
}

// sendMonitorIncidents sends the recent incident history
// for the selected monitor.
func (b *Bot) sendMonitorIncidents(ctx context.Context, chatID int64, target *domain.Target) error {
	incidents, err := b.listIncidents.ListIncidents(ctx, target.ID, 5)
	if err != nil {
		return fmt.Errorf("list incidents: %w", err)
	}

	var response strings.Builder

	if len(incidents) == 0 {
		fmt.Fprintf(
			&response,
			"✅ No incidents recorded for monitor #%d.\n",
			target.ID,
		)
	} else {
		fmt.Fprintf(
			&response,
			"🚨 Incidents for monitor #%d\nAddress: %s\n\n",
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
					incident.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
					formatObservedDuration(time.Since(incident.CreatedAt)),
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
				incident.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
				incident.ClosedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
				formatObservedDuration(incident.Duration()),
			)
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"‹ Back to monitor",
				fmt.Sprintf("monitor:%d", target.ID),
			),
		),
	)

	msg := tgbotapi.NewMessage(chatID, response.String())
	msg.ReplyMarkup = keyboard

	_, err = b.api.Send(msg)
	return err
}

// sendRemoveConfirmation asks the user to confirm
// deletion of the selected monitor.
func (b *Bot) sendRemoveConfirmation(ctx context.Context, chatID int64, target *domain.Target) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"✅ Yes, remove",
				fmt.Sprintf("remove_confirm:%d", target.ID),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"Cancel",
				fmt.Sprintf("monitor:%d", target.ID),
			),
		),
	)

	msg := tgbotapi.NewMessage(
		chatID,
		fmt.Sprintf(
			"🗑 Remove monitor #%d?\n\n%s\n\nThis action cannot be undone.",
			target.ID,
			target.Address,
		),
	)
	msg.ReplyMarkup = keyboard

	_, err := b.api.Send(msg)
	return err
}
