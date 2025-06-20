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
	case isCommand(text, "/start"):
		h.handleStart(chatID, user)

	case isCommand(text, "/athlete"):
		h.handleAthlete(chatID, update)

	case strings.HasPrefix(text, "/coach "):
		h.handleCoach(chatID, update)

	case isCommand(text, "/ranking"):
		h.handleRanking(chatID, user)

	case strings.HasPrefix(text, "/request"):
		h.handleRequest(chatID, text, user)

	case isCommand(text, "/pending"):
		h.handlePending(chatID, user)

	case strings.HasPrefix(text, "/approve "):
		h.handleApprove(chatID, text, user)

	case strings.HasPrefix(text, "/give "):
		h.handleGive(chatID, text, user)

	case isCommand(text, "/athletes"):
		h.handleAthletes(chatID, user)

	case strings.HasPrefix(text, "/reject "):
		h.handleReject(chatID, text, user)

	case strings.HasPrefix(text, "/history"):
		h.handleHistory(chatID, text, user)

	case isCommand(text, "/my_score"):
		h.handleMyScore(chatID, user)

	default:
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❓ Неизвестная команда. Напиши /start."))
	}
}

func (h *TelegramHandler) handleStart(chatID int64, user *domain.User) {
	if user == nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID,
			"👋 Привет! Напиши /athlete если ты спортсмен, или /coach <секретный_ключ> если ты тренер."))
	} else if user.Role == domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID,
			fmt.Sprintf("👋 Добро пожаловать, тренер %s!", user.Name)))
	} else {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID,
			fmt.Sprintf("👋 Привет, %s! Ты зарегистрирован как спортсмен.", user.Name)))
	}
}

func (h *TelegramHandler) handleAthlete(chatID int64, update tgbotapi.Update) {
	name := update.Message.From.FirstName
	username := update.Message.From.UserName
	err := h.Repo.RegisterUser(&domain.User{
		ID:       chatID,
		Name:     name,
		Username: username,
		Role:     domain.RoleAthlete,
	})
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка регистрации: "+err.Error()))
	} else {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "✅ Ты зарегистрирован как спортсмен."))
	}
}

func (h *TelegramHandler) handleCoach(chatID int64, update tgbotapi.Update) {
	providedKey := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/coach "))
	if providedKey != h.SecretCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Неверный секретный ключ."))
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

// handleRanking sends the leaderboard to the user.
func (h *TelegramHandler) handleRanking(chatID int64, user *domain.User) {
	if user == nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "Сначала зарегистрируйся через /start."))
		return
	}

	ranking, err := h.Repo.GetRanking()
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка при получении рейтинга: "+err.Error()))
		return
	}

	if len(ranking) == 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "Рейтинг пока пуст."))
		return
	}

	msg := "🏆 Рейтинг спортсменов:\n"
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
	if len(parts) < 3 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Формат: /request <баллы> <причина>"))
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

// handlePending lists all pending requests for the coach.
func (h *TelegramHandler) handlePending(chatID int64, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
		return
	}

	requests, err := h.Repo.GetPendingRequests()
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка при получении списка запросов: "+err.Error()))
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

// Lists all athletes for the coach
func (h *TelegramHandler) handleAthletes(chatID int64, user *domain.User) {
	if user == nil || user.Role != domain.RoleCoach {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
		return
	}

	athletes, err := h.Repo.ListAthletes()
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Не удалось получить список: "+err.Error()))
		return
	}

	if len(athletes) == 0 {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "📭 Спортсмены пока не зарегистрированы."))
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

	if len(args) == 1 {
		if user.Role != domain.RoleAthlete {
			util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❗ Тренеры должны указать @username для просмотра истории спортсмена."))
			return
		}
		targetID = chatID
		targetName = user.Username
	} else if len(args) == 2 && user.Role == domain.RoleCoach {
		username := strings.TrimPrefix(args[1], "@")
		targetUser, err := h.Repo.GetUserByUsername(username)
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

	history, err := h.Repo.GetUserHistory(targetID)
	if err != nil {
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, "❌ Ошибка при получении истории: "+err.Error()))
		return
	}

	if len(history) == 0 {
		msg := "📭 Нет начислений."
		if user.Role == domain.RoleCoach && targetID != chatID {
			msg = fmt.Sprintf("📭 У пользователя @%s пока нет начислений.", targetName)
		}
		util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
		return
	}

	msg := "📜 История начислений:\n\n"
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

	msg := fmt.Sprintf("🏅 Твой текущий счёт: %d баллов", score)
	util.SafeSend(h.Bot, tgbotapi.NewMessage(chatID, msg))
}
