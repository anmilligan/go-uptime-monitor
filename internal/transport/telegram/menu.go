package telegram

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"fmt"
	"net/url"

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
				"📊 List",
				"menu:list_full",
			),
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

// sendMonitorDetails sends the current monitor state
// together with available inline actions.
func (b *Bot) sendMonitorDetails(ctx context.Context, chatID int64, target *domain.Target) error {
	response, err := b.buildMonitorDetailsText(
		ctx,
		target,
	)
	if err != nil {
		return err
	}

	actionButton :=
		tgbotapi.NewInlineKeyboardButtonData(
			"⏸ Pause",
			fmt.Sprintf(
				"pause:%d",
				target.ID,
			),
		)

	if !target.Active {
		actionButton =
			tgbotapi.NewInlineKeyboardButtonData(
				"▶️ Resume",
				fmt.Sprintf(
					"resume:%d",
					target.ID,
				),
			)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🚨 Incidents",
				fmt.Sprintf(
					"incidents:%d",
					target.ID,
				),
			),
			actionButton,
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🗑 Remove",
				fmt.Sprintf(
					"remove:%d",
					target.ID,
				),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"‹ Back to monitors",
				"menu:list",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"⌂ Main menu",
				"menu:main",
			),
		),
	)

	msg := tgbotapi.NewMessage(
		chatID,
		response,
	)

	msg.ReplyMarkup = keyboard

	_, err = b.api.Send(msg)

	return err
}

// sendMonitorIncidents sends the recent incident history
// together with navigation back to the selected monitor.
func (b *Bot) sendMonitorIncidents(
	ctx context.Context,
	chatID int64,
	target *domain.Target,
) error {
	response, err := b.buildMonitorIncidentsText(
		ctx,
		target,
	)
	if err != nil {
		return err
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"‹ Back to monitor",
				fmt.Sprintf(
					"monitor:%d",
					target.ID,
				),
			),
		),
	)

	msg := tgbotapi.NewMessage(
		chatID,
		response,
	)

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
