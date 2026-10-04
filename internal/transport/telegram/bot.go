package telegram

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/usecase"
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
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

func (b *Bot) Run(ctx context.Context) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 25

	updates := b.api.GetUpdatesChan(updateConfig)
	defer b.api.StopReceivingUpdates()

	log.Printf(
		"telegram bot started: @%s",
		b.api.Self.UserName,
	)

	for {
		select {
		case <-ctx.Done():
			log.Println("telegram bot stopped")
			return

		case update, ok := <-updates:
			if !ok {
				return
			}

			if err := b.handleUpdate(ctx, update); err != nil {
				log.Printf(
					"telegram: failed to handle update %d",
					update.UpdateID,
				)
			}
		}
	}
}

func (b *Bot) handleUpdate(
	ctx context.Context,
	update tgbotapi.Update,
) error {
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

		log.Printf(
			"telegram: registered user %d",
			user.ID,
		)

		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"✅ Welcome to Uptime Monitor!\n"+
				"Registration is complete. You can now "+
				"receive notifications about your monitors.\n\n"+
				"Use /help to see the available commands.",
		)

	case "help":
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"📖 Uptime Monitor — Help\n\n"+
				"/start — Register and start using the bot\n"+
				"/help — Show this help message\n"+
				"/add <url> [interval] — Add an HTTP monitor\n"+
				"/list — Show your monitors\n"+
				"/status <id> — Show monitor details\n\n"+
				"/incidents <id> — Show recent incidents\n"+
				"/remove <id> — Remove a monitor\n"+
				"/pause <id> — Pause monitor checks\n"+
				"/resume <id> — Resume monitor checks\n"+
				"More monitor management commands are coming soon.",
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

func (b *Bot) handleAdd(
	ctx context.Context,
	message *tgbotapi.Message,
) error {
	args := strings.Fields(message.CommandArguments())

	// /add без адреса или со слишком большим
	// количеством параметров.
	if len(args) == 0 || len(args) > 2 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
			"Usage: /add https://example.com 60\n"+
				"Interval is optional (default: 60 seconds).",
		)
	}

	address := args[0]

	// Проверяем, что пользователь передал HTTP(S)-адрес.
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

	// Интервал по умолчанию.
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

	// Получаем внутренний ID пользователя.
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

	// Создаём монитор через существующий use case.
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
		log.Printf("telegram: failed to add monitor: %v", err)

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

func (b *Bot) handleList(
	ctx context.Context,
	message *tgbotapi.Message,
) error {
	user, err := b.register.Register(
		ctx,
		message.From.ID,
		message.From.UserName,
	)
	if err != nil {
		return fmt.Errorf("register user for /list: %w", err)
	}

	targets, err := b.listTargets.ListTargets(
		ctx,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("list user monitors: %w", err)
	}

	if len(targets) == 0 {
		return b.notifier.Send(
			ctx,
			message.Chat.ID,
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

		// Показываем, участвует ли монитор
		// в плановых проверках.
		if target.Active {
			response.WriteString(
				"Monitoring: ▶️ ACTIVE\n",
			)
		} else {
			response.WriteString(
				"Monitoring: ⏸ PAUSED\n",
			)
		}

		// Получаем последнюю проверку текущего монитора.
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

		// HTTP-код показываем только для HTTP-мониторов.
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
		message.Chat.ID,
		response.String(),
	)
}

func (b *Bot) handleStatus(
	ctx context.Context,
	message *tgbotapi.Message,
) error {
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

	// Получаем внутренний ID пользователя.
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

	// Ищем монитор только среди принадлежащих пользователю.
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

	// Получаем последнюю проверку.
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

	// Получаем открытый инцидент, если он есть.
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

	// Считаем uptime за последние 24 часа.
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

	// Считаем uptime за последние 7 дней.
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

	// Определяем текущее состояние монитора.
	// PAUSED имеет приоритет, потому что монитор сейчас
	// не участвует в плановых проверках.
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
	// Информация о последней проверке.
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

	// Uptime за 24 часа и 7 дней.
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
	// Информация об открытом инциденте.
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

func (b *Bot) handleIncidents(
	ctx context.Context,
	message *tgbotapi.Message,
) error {
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

	// Получаем внутренний ID пользователя.
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

	// Проверяем, что монитор действительно принадлежит этому пользователю.
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

	// Получаем последние 5 инцидентов.
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

func (b *Bot) handleRemove(
	ctx context.Context,
	message *tgbotapi.Message,
) error {
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

	// Получаем только monitors текущего пользователя.
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

	// Не даём удалить чужой или несуществующий monitor.
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

func (b *Bot) handlePause(
	ctx context.Context,
	message *tgbotapi.Message,
) error {
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

	// Получаем внутренний ID пользователя.
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

	// Получаем только monitors текущего пользователя.
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

	// Не выполняем лишний UPDATE, если monitor уже на паузе.
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

func (b *Bot) handleResume(
	ctx context.Context,
	message *tgbotapi.Message,
) error {
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

	// Получаем внутренний ID пользователя.
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

	// Paused monitor всё ещё возвращается через ListByUserID(),
	// поэтому пользователь может снова его активировать.
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
