package telegram

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/usecase"
	"context"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type IRegisterTelegramUserUseCase interface {
	Register(ctx context.Context, telegramID int64, username string) (*domain.User, error)
}

type INotifier interface {
	Send(ctx context.Context, chatID int64, message string) error
}
type IAddTargetUseCase interface {
	AddTarget(ctx context.Context, req usecase.AddTargetRequest) (*domain.Target, error)
}

type IListTargetsUseCase interface {
	ListTargets(ctx context.Context, userID int) ([]domain.Target, error)
}

type IGetLastCheckUseCase interface {
	GetLastCheck(ctx context.Context, targetID int) (*domain.Check, error)
}

type IGetOpenIncidentUseCase interface {
	GetOpenIncident(ctx context.Context, targetID int) (*domain.Incident, error)
}

type ICalculateUptimeUseCase interface {
	CalculateUptime(ctx context.Context, targetID int, targetCreatedAt time.Time, period time.Duration) (*usecase.UptimeResult, error)
}

type IListIncidentsUseCase interface {
	ListIncidents(ctx context.Context, targetID int, limit int) ([]domain.Incident, error)
}

type IDeleteTargetUseCase interface {
	DeleteTarget(ctx context.Context, id int) error
}

type IPauseTargetUseCase interface {
	PauseTarget(ctx context.Context, id int) error
}

type IResumeTargetUseCase interface {
	ResumeTarget(ctx context.Context, id int) error
}

// Bot handles incoming Telegram updates and routes user actions
// to the corresponding application use cases.
type Bot struct {
	api           *tgbotapi.BotAPI
	register      IRegisterTelegramUserUseCase
	addTarget     IAddTargetUseCase
	listTargets   IListTargetsUseCase
	lastCheck     IGetLastCheckUseCase
	openIncident  IGetOpenIncidentUseCase
	uptime        ICalculateUptimeUseCase
	listIncidents IListIncidentsUseCase
	deleteTarget  IDeleteTargetUseCase
	pauseTarget   IPauseTargetUseCase
	resumeTarget  IResumeTargetUseCase
	notifier      INotifier
}

// NewBot creates a Telegram bot with the required application dependencies.
func NewBot(
	api *tgbotapi.BotAPI,
	register IRegisterTelegramUserUseCase,
	addTarget IAddTargetUseCase,
	listTargets IListTargetsUseCase,
	lastCheck IGetLastCheckUseCase,
	openIncident IGetOpenIncidentUseCase,
	uptime ICalculateUptimeUseCase,
	listIncidents IListIncidentsUseCase,
	deleteTarget IDeleteTargetUseCase,
	pauseTarget IPauseTargetUseCase,
	resumeTarget IResumeTargetUseCase,
	notifier INotifier,
) *Bot {
	return &Bot{
		api:           api,
		register:      register,
		addTarget:     addTarget,
		listTargets:   listTargets,
		lastCheck:     lastCheck,
		openIncident:  openIncident,
		uptime:        uptime,
		listIncidents: listIncidents,
		deleteTarget:  deleteTarget,
		pauseTarget:   pauseTarget,
		resumeTarget:  resumeTarget,
		notifier:      notifier,
	}
}

// Run starts receiving Telegram updates and stops when the context is canceled.
func (b *Bot) Run(ctx context.Context) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 25

	updates := b.api.GetUpdatesChan(updateConfig)
	defer b.api.StopReceivingUpdates()

	slog.Info(
		"telegram bot started",
		"username", b.api.Self.UserName,
	)

	for {
		select {
		case <-ctx.Done():
			slog.Info("telegram bot stopped")
			return

		case update, ok := <-updates:
			if !ok {
				return
			}

			if err := b.handleUpdate(ctx, update); err != nil {
				slog.ErrorContext(
					ctx,
					"telegram failed to handle update",
					"update_id", update.UpdateID,
					"error", err,
				)
			}
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	if update.CallbackQuery != nil {
		return b.handleCallback(
			ctx,
			update.CallbackQuery,
		)
	}
	message := update.Message

	// Обрабатываем только сообщения от реальных
	// пользователей в личном чате с ботом.
	if message == nil ||
		message.From == nil ||
		message.From.IsBot ||
		message.Chat == nil ||
		message.Chat.Type != "private" {
		return nil
	}

	switch message.Command() {

	case "start":
		user, err := b.register.Register(
			ctx,
			message.From.ID,
			message.From.UserName,
		)
		if err != nil {
			return err
		}

		slog.InfoContext(
			ctx,
			"telegram user registered",
			"user_id", user.ID,
		)

		return b.sendMainMenu(
			ctx,
			message.Chat.ID,
		)

	case "help":
		return b.sendHelp(
			ctx,
			message.Chat.ID,
		)

	case "add":
		return b.handleAdd(ctx, message)

	case "list":
		return b.handleList(ctx, message)

	case "status":
		return b.handleStatus(ctx, message)

	case "incidents":
		return b.handleIncidents(ctx, message)

	case "remove":
		return b.handleRemove(ctx, message)

	case "pause":
		return b.handlePause(ctx, message)

	case "resume":
		return b.handleResume(ctx, message)

	default:
		return nil
	}
}
