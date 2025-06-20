// internal/handler/router.go
package handler

import (
	"fmt"
	"strconv"
	"strings"

	"surf_bot/internal/domain"
	"surf_bot/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func isCommand(text, cmd string) bool {
	return strings.TrimSpace(text) == cmd
}

func (h *TelegramHandler) HandleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := update.Message.Text

	user, _ := h.Repo.GetUserByID(chatID)

	switch {
	case strings.HasPrefix(text, "/start"):
		args := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
    	h.handleStart(chatID, user, args, update.Message.From)

	case isCommand(text, "/athlete"):
		h.handleAthlete(chatID, update)

	case strings.HasPrefix(text, "/coach"):
		h.handleCoach(chatID, update)

	case strings.HasPrefix(text, "/ranking"):
		h.handleRanking(chatID, user, text)

	case strings.HasPrefix(text, "/request"):
		h.handleRequest(chatID, text, user)

	case strings.HasPrefix(text, "/pending"):
		h.handlePending(chatID, user, text)

	case strings.HasPrefix(text, "/approve"):
		h.handleApprove(chatID, text, user)

	case strings.HasPrefix(text, "/give"):
		h.handleGive(chatID, text, user)

	case strings.HasPrefix(text, "/athletes"):
		h.handleAthletes(chatID, user, text)

	case strings.HasPrefix(text, "/reject"):
		h.handleReject(chatID, text, user)

	case strings.HasPrefix(text, "/history"):
		h.handleHistory(chatID, text, user)

	case isCommand(text, "/my_score"):
		h.handleMyScore(chatID, user)
	
	case isCommand(text, "/teams"):
		h.handleTeams(chatID, user)
	
	case strings.HasPrefix(text, "/invite_link"):
		h.handleInviteLink(chatID, text, user)
	
	case strings.HasPrefix(text, "/create_team"):
		h.handleCreateTeam(chatID, text, user)

	case strings.HasPrefix(text, "/delete_team"):
		h.handleDeleteTeam(chatID, text, user)

	case strings.HasPrefix(text, "/assign_team"):
		h.handleAssignTeam(chatID, text, user)

	default:
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❓ Неизвестная команда. Напиши /start."))
	}
}

func (h *TelegramHandler) SetBotCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Начать работу с ботом"},
		{Command: "athlete", Description: "Зарегистрироваться как спортсмен"},
		{Command: "coach", Description: "Зарегистрироваться как тренер"},
		{Command: "ranking", Description: "Посмотреть рейтинг своей команды"},
		{Command: "request", Description: "Запросить баллы: /request <баллы> <причина>"},
		{Command: "pending", Description: "Список ожидающих запросов"},
		{Command: "approve", Description: "Подтвердить запрос: /approve <id>"},
		{Command: "reject", Description: "Отклонить запрос: /reject <id>"},
		{Command: "give", Description: "Начислить баллы: /give @username <баллы> <причина>"},
		{Command: "athletes", Description: "Список спортсменов"},
		{Command: "history", Description: "История начислений или /history @username"},
		{Command: "my_score", Description: "Текущий счёт спортсмена"},
		{Command: "create_team", Description: "Создать команду: /create_team <название>"},
		{Command: "delete_team", Description: "Удалить команду: /delete_team <название>"},
		{Command: "assign_team", Description: "Добавить в команду: /assign_team @username <team_id>"},
		{Command: "teams", Description: "Список всех команд"},
		{Command: "invite_link", Description: "Пригласить в команду: /invite_link <team_id>"},

	}

	cfg := tgbotapi.NewSetMyCommands(commands...)
	_, err := h.Bot.Request(cfg)
	return err
}

func (h *TelegramHandler) handleStart(chatID int64, user *domain.User, args string, from *tgbotapi.User) {
	if user == nil {
		if strings.HasPrefix(args, "team_") {
			teamIDStr := strings.TrimPrefix(args, "team_")
			teamID, err := strconv.Atoi(teamIDStr)
			if err == nil && teamID > 0 {
				// Проверка наличия команды
				team, err := h.Repo.GetTeamByID(teamID)
				if err == nil {
					newUser := &domain.User{
						ID:       chatID,
						Name:     from.FirstName,
						Username: from.UserName,
						Role:     domain.RoleAthlete,
					}
					err = h.Repo.RegisterUser(newUser)
					if err == nil {
						_ = h.Repo.AssignUserToTeam(chatID, teamID)
						util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID,
							fmt.Sprintf("✅ Ты зарегистрирован как спортсмен в команде '%s'.", team.Name)))
						return
					}
				}
			}

			util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID,
				"❌ Не удалось зарегистрироваться: неверный ID команды или ошибка регистрации."))
			return
		}

		msg := "👋 Привет! Добро пожаловать в SurfCoinBot.\n\n" +
			"Для начала укажи свою роль:\n" +
			"• /athlete — если ты спортсмен\n" +
			"• /coach <секретный_ключ> — если ты тренер\n\n" +
			"После этого ты сможешь использовать соответствующие команды."
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
		return
	}

	// уже зарегистрирован
	switch user.Role {
	case domain.RoleAthlete:
		msg := "👋 Привет, " + user.Name + "! Ты зарегистрирован как спортсмен.\n\n" +
			"📋 Доступные команды:\n" +
			"• /request <баллы> <причина> — отправить запрос на баллы\n" +
			"• /my_score — посмотреть свой счёт\n" +
			"• /ranking — общий рейтинг\n" +
			"• /history — история начислений\n"
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))

	case domain.RoleCoach:
		msg := "👋 Добро пожаловать, тренер " + user.Name + "!\n\n" +
			"📋 Доступные команды:\n" +
			"• /pending — список запросов на подтверждение\n" +
			"• /approve <id> — подтвердить запрос\n" +
			"• /reject <id> — отклонить запрос\n" +
			"• /give <баллы> @username <причина> — начислить баллы вручную\n" +
			"• /athletes — список спортсменов\n" +
			"• /ranking — рейтинг по командам\n" +
			"• /history @username — история начислений спортсмена\n" +
			"• /create_team <название> — создать команду\n" +
			"• /delete_team <название> — удалить команду\n" +
			"• /assign_team @username <team_id> — прикрепить спортсмена\n" +
			"• /teams — список команд\n" +
			"• /invite_link <team_id> — получить ссылку-приглашение"
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
	}
}


func (h *TelegramHandler) handleAthlete(chatID int64, update tgbotapi.Update) {
	args := strings.Fields(update.Message.Text)
	if len(args) != 2 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Формат: /athlete <id_команды>"))
		return
	}

	teamID, err := strconv.Atoi(args[1])
	if err != nil || teamID <= 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Неверный ID команды."))
		return
	}

	team, err := h.Repo.GetTeamByID(teamID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Команда не найдена: "+err.Error()))
		return
	}

	existing, _ := h.Repo.GetUserByID(chatID)
	if existing != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID,
			fmt.Sprintf("ℹ️ Ты уже зарегистрирован как %s.", existing.Role)))
		return
	}

	name := update.Message.From.FirstName
	username := update.Message.From.UserName

	user := &domain.User{
		ID:       chatID,
		Name:     name,
		Username: username,
		Role:     domain.RoleAthlete,
	}

	err = h.Repo.RegisterUser(user)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка регистрации: "+err.Error()))
		return
	}

	err = h.Repo.AssignUserToTeam(chatID, team.ID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось добавить в команду: "+err.Error()))
		return
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID,
		fmt.Sprintf("✅ Ты зарегистрирован как спортсмен в команде '%s'.", team.Name)))
}


func (h *TelegramHandler) handleCoach(chatID int64, update tgbotapi.Update) {
	providedKey := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/coach "))
	if providedKey != h.SecretCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Неверный секретный ключ."))
		return
	}

	user, _ := h.Repo.GetUserByID(chatID)
	if user != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "ℹ️ Ты уже зарегистрирован как "+string(user.Role)+"."))
		return
	}

	name := update.Message.From.FirstName
	username := update.Message.From.UserName
	err := h.Repo.RegisterUser(&domain.User{
		ID:       chatID,
		Name:     name,
		Username: username,
		Role:     domain.RoleCoach,
	})
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка регистрации: "+err.Error()))
	} else {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "✅ Ты зарегистрирован как тренер."))
	}
}

func (h *TelegramHandler) handleRanking(chatID int64, user *domain.User, text string) {
	if user == nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "Сначала зарегистрируйся через /start."))
		return
	}

	var (
		teamID   int
		teamName string
	)

	if strings.Contains(text, "team:") {
		parts := strings.Fields(text)
		for _, part := range parts {
			if strings.HasPrefix(part, "team:") {
				idStr := strings.TrimPrefix(part, "team:")
				id, err := strconv.Atoi(idStr)
				if err == nil {
					teamID = id
				}
			}
		}
	}

	var ranking []domain.ScoreEntry
	var err error

	if teamID > 0 {
		team, errTeam := h.Repo.GetTeamByID(teamID)
		if errTeam != nil {
			util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Команда не найдена."))
			return
		}
		teamName = team.Name

		ranking, err = h.Repo.GetRankingByTeam(teamID)
	} else {
		ranking, err = h.Repo.GetRanking()
	}

	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка при получении рейтинга: "+err.Error()))
		return
	}

	if len(ranking) == 0 {
		msg := "Рейтинг пока пуст."
		if teamID > 0 {
			msg = fmt.Sprintf("В команде %s пока нет спортсменов с баллами.", teamName)
		}
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
		return
	}

	title := "🏆 Общий рейтинг спортсменов:\n"
	if teamID > 0 {
		title = fmt.Sprintf("🏆 Рейтинг команды %s:\n", teamName)
	}

	msg := title
	var userRankText string
	for i, r := range ranking {
		msg += fmt.Sprintf("%d. %s (@%s) — %d баллов\n", i+1, r.Name, r.Username, r.Score)
		if r.UserID == chatID {
			userRankText = fmt.Sprintf("\n📍 Ты на %d месте с %d баллами.", i+1, r.Score)
		}
	}
	msg += userRankText

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
}



// handleRequest processes an athlete's points request.
func (h *TelegramHandler) handleRequest(chatID int64, text string, user *domain.User) {
	if user == nil || user.Role != domain.RoleAthlete {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Только спортсмены могут отправлять запросы на баллы."))
		return
	}

	// Parse command: /request <amount> <reason>
	parts := strings.SplitN(text, " ", 3)
	if len(parts) < 2 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи количество баллов после команды /request."))
		return
	}
	if len(parts) < 3 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи причину запроса после количества баллов."))
		return
	}

	amount, err := strconv.Atoi(parts[1])
	if err != nil || amount <= 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректное число баллов больше нуля."))
		return
	}

	reason := strings.TrimSpace(parts[2])
	if reason == "" {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи причину запроса."))
		return
	}

	err = h.Repo.CreatePendingRequest(chatID, amount, reason)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось создать запрос: "+err.Error()))
		return
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("📨 Запрос на %d баллов отправлен на подтверждение тренеру.", amount)))
}

func (h *TelegramHandler) handlePending(chatID int64, user *domain.User, text string) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
		return
	}

	args := strings.Fields(text)
	var teamID *int = nil
	if len(args) == 2 {
		id, err := strconv.Atoi(args[1])
		if err != nil || id <= 0 {
			util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректный ID команды."))
			return
		}
		teamID = &id
	}

	requests, err := h.Repo.GetPendingRequestsByTeam(teamID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка при получении запросов: "+err.Error()))
		return
	}

	if len(requests) == 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "✅ Нет ожидающих запросов."))
		return
	}

	msg := "📋 Ожидающие запросы:\n"
	for _, req := range requests {
		msg += fmt.Sprintf("ID: %d | 👤 %s (@%s) | ➕ %d баллов\n📎 %s\n\n",
			req.ID, req.Name, req.Username, req.Amount, req.Reason)
	}
	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
}

// Approves a pending point request
func (h *TelegramHandler) handleApprove(chatID int64, text string, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
		return
	}

	idStr := strings.TrimSpace(strings.TrimPrefix(text, "/approve "))
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректный ID запроса."))
		return
	}

	err = h.Repo.ApproveRequest(id)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось подтвердить запрос: "+err.Error()))
		return
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Запрос #%d подтвержден. Баллы начислены.", id)))
}

// Gives points directly to an athlete
func (h *TelegramHandler) handleGive(chatID int64, text string, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Только тренеры могут использовать эту команду."))
		return
	}

	args := strings.SplitN(strings.TrimPrefix(text, "/give "), " ", 3)
	if len(args) < 3 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Формат: /give <баллы> @имя <причина>"))
		return
	}

	amount, err := strconv.Atoi(args[0])
	if err != nil || amount <= 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректное количество баллов."))
		return
	}

	username := strings.TrimPrefix(args[1], "@")
	reason := strings.TrimSpace(args[2])

	err = h.Repo.GivePoints(username, amount, reason)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка: "+err.Error()))
		return
	}

	msg := fmt.Sprintf("✅ %d баллов начислены пользователю @%s.\n📎 %s", amount, username, reason)
	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
}

// Updated handler for listing athletes with optional team ID
func (h *TelegramHandler) handleAthletes(chatID int64, user *domain.User, text string) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
		return
	}

	args := strings.Fields(text)
	var (
		teamID *int = nil
		athletes []domain.AthleteShort
		err error
	)

	if len(args) == 2 {
		id, convErr := strconv.Atoi(args[1])
		if convErr != nil || id <= 0 {
			util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректный ID команды."))
			return
		}
		teamID = &id
	}

	athletes, err = h.Repo.ListAthletesByTeam(teamID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось получить список: "+err.Error()))
		return
	}

	if len(athletes) == 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "📭 В этой команде пока нет спортсменов."))
		return
	}

	msg := "📋 Список спортсменов:\n\n"
	for _, a := range athletes {
		msg += fmt.Sprintf("• @%s (%s)\n", a.Username, a.Name)
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
}

func (h *TelegramHandler) handleReject(chatID int64, text string, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
		return
	}

	idStr := strings.TrimSpace(strings.TrimPrefix(text, "/reject "))
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректный ID запроса."))
		return
	}

	fromID, err := h.Repo.RejectRequest(id)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось отклонить запрос: "+err.Error()))
		return
	}

	// Уведомление спортсмену
	util.SafeSend(h.Bot, tgbotapi.NewMessage(fromID, fmt.Sprintf("🚫 Ваш запрос #%d на баллы был отклонён тренером.", id)))

	// Подтверждение тренеру
	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("🚫 Запрос #%d отклонён и удалён.", id)))
}

func (h *TelegramHandler) handleHistory(chatID int64, text string, user *domain.User) {
	if user == nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Пользователь не найден."))
		return
	}

	args := strings.Fields(text)
	var targetID int64
	var targetName string
	var targetUser *domain.User

	if len(args) == 1 {
		if user.Role != domain.RoleAthlete {
			util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Тренеры должны указать @username для просмотра истории спортсмена."))
			return
		}
		targetID = chatID
		targetUser = user
	} else if len(args) == 2 && user.Role == domain.RoleCoach {
		username := strings.TrimPrefix(args[1], "@")
		var err error
		targetUser, err = h.Repo.GetUserByUsername(username)
		if err != nil || targetUser == nil || targetUser.Role != domain.RoleAthlete {
			util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Спортсмен с таким username не найден."))
			return
		}
		targetID = targetUser.ID
		targetName = targetUser.Username
	} else {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Формат: /history или /history @username (для тренера)"))
		return
	}

	// Получаем историю начислений
	history, err := h.Repo.GetUserHistory(targetID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка при получении истории: "+err.Error()))
		return
	}

	// Получаем команду
	teamName := ""
	if targetUser != nil {
		teamName, err = h.Repo.GetUserTeamName(targetUser.ID)
		if err != nil {
			teamName = ""
		}
	}

	if len(history) == 0 {
		msg := "📭 Нет начислений."
		if user.Role == domain.RoleCoach && targetID != chatID {
			msg = fmt.Sprintf("📭 У пользователя @%s пока нет начислений.", targetName)
		}
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
		return
	}

	msg := "📜 История начислений"
	if teamName != "" {
		msg += fmt.Sprintf(" (команда: %s)", teamName)
	}
	msg += ":\n\n"

	for _, entry := range history {
		msg += fmt.Sprintf("• ➕ %d баллов — %s\n", entry.Amount, entry.Reason)
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
}


func (h *TelegramHandler) handleMyScore(chatID int64, user *domain.User) {
	if user == nil || user.Role != domain.RoleAthlete {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только спортсменам."))
		return
	}

	score, err := h.Repo.GetUserScore(chatID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка при получении счёта: "+err.Error()))
		return
	}

	teamID, err := h.Repo.GetUserTeamID(chatID)
	if err != nil || teamID == 0 {
		msg := fmt.Sprintf("🏅 Твой текущий счёт: %d баллов\n\n📌 Ты не прикреплён ни к одной команде.", score)
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
		return
	}

	// Получаем рейтинг по команде
	ranking, err := h.Repo.GetRankingByTeam(teamID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка при получении рейтинга команды: "+err.Error()))
		return
	}

	place := -1
	for i, entry := range ranking {
		if entry.UserID == chatID {
			place = i + 1
			break
		}
	}

	msg := fmt.Sprintf("🏅 Твой текущий счёт: %d баллов", score)
	if place > 0 {
		msg += fmt.Sprintf("\n📊 Ты на %d месте в своей команде.", place)
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
}


func (h *TelegramHandler) handleTeams(chatID int64, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
		return
	}

	teams, err := h.Repo.ListTeams()
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось получить список команд: "+err.Error()))
		return
	}

	if len(teams) == 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "📭 Пока нет ни одной команды."))
		return
	}

	msg := "📦 Список команд:\n\n"
	for _, t := range teams {
		msg += fmt.Sprintf("• %s (ID: %d)\n", t.Name, t.ID)
	}
	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
}

func (h *TelegramHandler) handleInviteLink(chatID int64, text string, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
		return
	}

	args := strings.Fields(text)
	if len(args) != 2 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Формат: /invite_link <team_id>"))
		return
	}

	teamID, err := strconv.Atoi(args[1])
	if err != nil || teamID <= 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректный числовой team_id."))
		return
	}

	link := fmt.Sprintf("https://t.me/%s?start=team_%d", h.Bot.Self.UserName, teamID)

	msg := fmt.Sprintf("🔗 Ссылка для приглашения в команду #%d:\n%s", teamID, link)

	button := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Присоединиться", link),
		),
	)

	message := tgbotapi.NewMessage(chatID, msg)
	message.ReplyMarkup = button

	util.SafeSend(h.Bot, message)
}


func (h *TelegramHandler) handleCreateTeam(chatID int64, text string, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Только тренеры могут создавать команды."))
		return
	}

	args := strings.Fields(text)
	if len(args) <= 2 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Формат: /create_team <название>"))
		return
	}

	name := strings.Join(args[1:], " ")
	err := h.Repo.CreateTeam(name)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось создать команду: "+err.Error()))
		return
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Команда \"%s\" создана.", name)))
}

func (h *TelegramHandler) handleDeleteTeam(chatID int64, text string, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Только тренеры могут удалять команды."))
		return
	}

	args := strings.Fields(text)
	if len(args) != 2 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Формат: /delete_team <id команды>"))
		return
	}

	teamID, err := strconv.Atoi(args[1])
	if err != nil || teamID <= 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректный team_id."))
		return
	}
	err = h.Repo.DeleteTeam(teamID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось удалить команду: "+err.Error()))
		return
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("🗑 Команда удалена.")))
}

func (h *TelegramHandler) handleAssignTeam(chatID int64, text string, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Только тренеры могут назначать команду."))
		return
	}

	args := strings.Fields(text)
	if len(args) <= 3 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Формат: /assign_team @username <team_id>"))
		return
	}

	username := strings.TrimPrefix(args[1], "@")
	teamID, err := strconv.Atoi(args[2])
	if err != nil || teamID <= 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Укажи корректный team_id."))
		return
	}

	athlete, err := h.Repo.GetUserByUsername(username)
	if err != nil || athlete == nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Пользователь не найден."))
		return
	}
	if athlete.Role != domain.RoleAthlete {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Только спортсменов можно добавлять в команды."))
		return
	}

	err = h.Repo.AssignUserToTeam(athlete.ID, teamID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось назначить команду: "+err.Error()))
		return
	}

	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID,
		fmt.Sprintf("✅ Пользователь @%s добавлен в команду #%d.", username, teamID)))
}
