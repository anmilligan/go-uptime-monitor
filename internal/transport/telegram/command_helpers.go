package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// parseMonitorIDArgument parses a positive monitor ID
// from a Telegram command argument.
func parseMonitorIDArgument(message *tgbotapi.Message, command string) (int, string) {
	args := strings.Fields(
		message.CommandArguments(),
	)

	if len(args) != 1 {
		return 0, fmt.Sprintf(
			"Usage: /%s <monitor_id>",
			command,
		)
	}

	targetID, err := strconv.Atoi(args[0])
	if err != nil || targetID <= 0 {
		return 0,
			"❌ Monitor ID must be a positive number."
	}

	return targetID, ""
}
