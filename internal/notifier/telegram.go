package notifier

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramNotifier struct {
	api *tgbotapi.BotAPI
}

func NewTelegramNotifier(
	api *tgbotapi.BotAPI,
) *TelegramNotifier {
	return &TelegramNotifier{
		api: api,
	}
}

func (n *TelegramNotifier) Send(
	ctx context.Context,
	chatID int64,
	message string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(chatID, message)

	_, err := n.api.Send(msg)
	if err != nil {
		return fmt.Errorf(
			"telegram send failed: %w",
			err,
		)
	}

	return nil
}
