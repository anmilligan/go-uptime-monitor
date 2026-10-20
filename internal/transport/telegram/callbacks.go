package telegram

import (
	"Uptime_Monitor/internal/domain"
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCallback routes inline button actions and verifies
// that monitor-specific actions target a monitor owned by the user.
func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) error {
	if callback == nil {
		return nil
	}

	// Telegram displays a loading indicator after an inline button is pressed.
	// We call the callback to make this indicator disappear immediately.
	callbackResponse := tgbotapi.NewCallback(
		callback.ID,
		"",
	)

	if _, err := b.api.Request(callbackResponse); err != nil {
		return fmt.Errorf(
			"answer callback query: %w",
			err,
		)
	}

	// Only process button clicks from a real user
	// in a private chat with the bot.
	if callback.From == nil ||
		callback.From.IsBot ||
		callback.Message == nil ||
		callback.Message.Chat == nil ||
		callback.Message.Chat.Type != "private" {
		return nil
	}

	chatID := callback.Message.Chat.ID
	telegramUserID := callback.From.ID
	username := callback.From.UserName

	switch callback.Data {
	case "menu:main":
		return b.sendMainMenu(ctx, chatID)

	case "menu:list":
		return b.sendMonitorSelection(
			ctx,
			chatID,
			telegramUserID,
			username,
		)

	case "menu:help":
		return b.sendHelp(ctx, chatID)

	case "menu:add":
		return b.notifier.Send(
			ctx,
			chatID,
			"➕ Add monitor\n\n"+
				"Send the command in this format:\n"+
				"/add https://example.com 60\n\n"+
				"Interval is optional. Default: 60 seconds.",
		)
	}

	action, targetID, ok := parseMonitorCallback(callback.Data)
	if !ok {
		return nil
	}

	target, err := b.getOwnedTarget(
		ctx,
		telegramUserID,
		username,
		targetID,
	)
	if err != nil {
		return err
	}

	if target == nil {
		return b.notifier.Send(
			ctx,
			chatID,
			"❌ Monitor not found or access denied.",
		)
	}

	switch action {
	case "monitor":
		return b.sendMonitorDetails(ctx, chatID, target)

	case "incidents":
		return b.sendMonitorIncidents(ctx, chatID, target)

	case "pause":
		if !target.Active {
			return b.notifier.Send(
				ctx,
				chatID,
				fmt.Sprintf("⏸ Monitor #%d is already paused.", target.ID),
			)
		}

		if err := b.pauseTarget.PauseTarget(ctx, target.ID); err != nil {
			return fmt.Errorf("pause monitor: %w", err)
		}

		target.Active = false

		if err := b.notifier.Send(
			ctx,
			chatID,
			fmt.Sprintf("⏸ Monitor #%d paused.\n%s", target.ID, target.Address),
		); err != nil {
			return err
		}

		return b.sendMonitorDetails(ctx, chatID, target)

	case "resume":
		if target.Active {
			return b.notifier.Send(
				ctx,
				chatID,
				fmt.Sprintf("▶️ Monitor #%d is already active.", target.ID),
			)
		}

		if err := b.resumeTarget.ResumeTarget(ctx, target.ID); err != nil {
			return fmt.Errorf("resume monitor: %w", err)
		}

		target.Active = true

		if err := b.notifier.Send(
			ctx,
			chatID,
			fmt.Sprintf("▶️ Monitor #%d resumed.\n%s", target.ID, target.Address),
		); err != nil {
			return err
		}

		return b.sendMonitorDetails(ctx, chatID, target)

	case "remove":
		return b.sendRemoveConfirmation(ctx, chatID, target)

	case "remove_confirm":
		if err := b.deleteTarget.DeleteTarget(ctx, target.ID); err != nil {
			return fmt.Errorf("delete monitor: %w", err)
		}

		if err := b.notifier.Send(
			ctx,
			chatID,
			fmt.Sprintf("✅ Monitor #%d removed.\n%s", target.ID, target.Address),
		); err != nil {
			return err
		}

		return b.sendMonitorSelection(
			ctx,
			chatID,
			telegramUserID,
			username,
		)

	default:
		return nil
	}
}

// parseMonitorCallback extracts the action and target ID
// from callback data such as "pause:12".
func parseMonitorCallback(data string) (string, int, bool) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return "", 0, false
	}

	targetID, err := strconv.Atoi(parts[1])
	if err != nil || targetID <= 0 {
		return "", 0, false
	}

	switch parts[0] {
	case "monitor", "incidents", "pause", "resume", "remove", "remove_confirm":
		return parts[0], targetID, true
	default:
		return "", 0, false
	}
}

// getOwnedTarget returns the requested target only if it belongs
// to the current Telegram user.
func (b *Bot) getOwnedTarget(ctx context.Context, telegramUserID int64, username string, targetID int) (*domain.Target, error) {
	user, err := b.register.Register(
		ctx,
		telegramUserID,
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("register telegram user: %w", err)
	}

	targets, err := b.listTargets.ListTargets(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("list user monitors: %w", err)
	}

	for i := range targets {
		if targets[i].ID == targetID {
			return &targets[i], nil
		}
	}

	return nil, nil
}
