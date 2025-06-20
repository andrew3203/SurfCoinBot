package handler

import (
	"surf_bot/internal/domain"
	"surf_bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramHandler struct {
	Repo        *repository.UserRepository
	SecretCoach string
	Bot         *tgbotapi.BotAPI
}

func NewTelegramHandler(repo *repository.UserRepository, bot *tgbotapi.BotAPI, secret string) *TelegramHandler {
	return &TelegramHandler{
		Repo:        repo,
		SecretCoach: secret,
		Bot:         bot,
	}
}

func (h *TelegramHandler) HandleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := update.Message.Text
	name := update.Message.From.FirstName
	username := update.Message.From.UserName,

	switch {
	case text == "/start":
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "👋 Привет! Напиши /athlete если ты спортсмен, или /coach <секретный_ключ> если ты тренер."))
		} else if user.Role == domain.RoleCoach {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "👋 Добро пожаловать, тренер "+user.Name+"!"))
		} else {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "👋 Привет, "+user.Name+"! Ты зарегистрирован как спортсмен."))
		}

	case text == "/athlete":
		err := h.Repo.RegisterUser(&domain.User{ID: chatID, Name: name, Role: domain.RoleAthlete, Username: username})
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка регистрации: "+err.Error()))
		} else {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "✅ Ты зарегистрирован как спортсмен."))
		}

	case len(text) > 7 && text[:7] == "/coach ":
		providedKey := text[7:]
		if providedKey != h.SecretCoach {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Неверный секретный ключ."))
			return
		}

		err := h.Repo.RegisterUser(&domain.User{ID: chatID, Name: name, Role: domain.RoleCoach, Username: username})
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка регистрации: "+err.Error()))
		} else {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "✅ Ты зарегистрирован как тренер."))
		}
	case text == "/ranking":
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "Сначала зарегистрируйся через /start."))
			return
		}

		ranking, err := h.Repo.GetRanking()
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении рейтинга: "+err.Error()))
			return
		}

		if len(ranking) == 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "Рейтинг пока пуст."))
			return
		}

		msg := "🏆 Рейтинг спортсменов:\n"
		var userRankText string

		for i, r := range ranking {
			line := fmt.Sprintf("%d. %s — %d баллов\n", i+1, r.Name, r.Score)
			msg += line

			if r.UserID == chatID {
				userRankText = fmt.Sprintf("\n📍 Ты на %d месте с %d баллами.", i+1, r.Score)
			}
		}
		msg += userRankText
		h.Bot.Send(tgbotapi.NewMessage(chatID, msg))
	
	case strings.HasPrefix(text, "/request "):
		args := strings.SplitN(text[len("/request "):], " ", 2)
		if len(args) < 2 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Формат: /request <баллы> <причина>"))
			return
		}

		amount, err := strconv.Atoi(args[0])
		if err != nil || amount <= 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Укажи корректное число баллов больше нуля."))
			return
		}
		reason := strings.TrimSpace(args[1])
		if reason == "" {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Укажи причину запроса."))
			return
		}

		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil || user.Role != domain.RoleAthlete {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Только спортсмены могут отправлять запросы на баллы."))
			return
		}

		err = h.Repo.CreatePendingRequest(chatID, amount, reason)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось создать запрос: "+err.Error()))
			return
		}

		msg := fmt.Sprintf("📨 Запрос на %d баллов отправлен на подтверждение тренеру.", amount)
		h.Bot.Send(tgbotapi.NewMessage(chatID, msg))
	case text == "/pending":
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil || user.Role != domain.RoleCoach {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
			return
		}

		requests, err := h.Repo.GetPendingRequests()
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка при получении списка: "+err.Error()))
			return
		}

		if len(requests) == 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "✅ Нет ожидающих запросов."))
			return
		}

		msg := "📋 Ожидающие запросы:\n"
		for _, req := range requests {
			line := fmt.Sprintf("ID: %d | 👤 %s | ➕ %d баллов\n📎 %s\n\n", req.ID, req.Name, req.Amount, req.Reason)
			msg += line
		}
		h.Bot.Send(tgbotapi.NewMessage(chatID, msg))
	
	case strings.HasPrefix(text, "/approve "):
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil || user.Role != domain.RoleCoach {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
			return
		}

		idStr := strings.TrimSpace(strings.TrimPrefix(text, "/approve "))
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Укажи корректный ID запроса."))
			return
		}

		err = h.Repo.ApproveRequest(id)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось подтвердить запрос: "+err.Error()))
			return
		}

		h.Bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Запрос #%d подтвержден. Баллы начислены.", id)))
	
	case strings.HasPrefix(text, "/give "):
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil || user.Role != domain.RoleCoach {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Только тренеры могут использовать эту команду."))
			return
		}

		args := strings.SplitN(strings.TrimPrefix(text, "/give "), " ", 3)
		if len(args) < 3 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Формат: /give <баллы> @имя <причина>"))
			return
		}

		amount, err := strconv.Atoi(args[0])
		if err != nil || amount <= 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Укажи корректное количество баллов."))
			return
		}

		username := strings.TrimPrefix(args[1], "@")
		reason := args[2]

		err = h.Repo.GivePoints(username, amount, reason)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка: "+err.Error()))
			return
		}

		msg := fmt.Sprintf("✅ %d баллов начислены пользователю @%s.\n📎 %s", amount, username, reason)
		h.Bot.Send(tgbotapi.NewMessage(chatID, msg))
	
	case text == "/athletes":
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil || user.Role != domain.RoleCoach {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
			return
		}

		athletes, err := h.Repo.ListAthletes()
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось получить список: "+err.Error()))
			return
		}

		if len(athletes) == 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "📭 Спортсмены пока не зарегистрированы."))
			return
		}

		msg := "📋 Список спортсменов:\n\n"
		for _, a := range athletes {
			msg += fmt.Sprintf("• @%s (%s)\n", a.Username, a.Name)
		}

		h.Bot.Send(tgbotapi.NewMessage(chatID, msg))
	
	case strings.HasPrefix(text, "/reject "):
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil || user.Role != domain.RoleCoach {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
			return
		}

		idStr := strings.TrimSpace(strings.TrimPrefix(text, "/reject "))
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Укажи корректный ID запроса."))
			return
		}

		err = h.Repo.RejectRequest(id)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось отклонить запрос: "+err.Error()))
			return
		}

		h.Bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🚫 Запрос #%d отклонён.", id)))

	case strings.HasPrefix(text, "/reject "):
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil || user.Role != domain.RoleCoach {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Команда доступна только тренерам."))
			return
		}

		idStr := strings.TrimSpace(strings.TrimPrefix(text, "/reject "))
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Укажи корректный ID запроса."))
			return
		}

		fromID, err := h.Repo.RejectRequest(id)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось отклонить запрос: "+err.Error()))
			return
		}

		// Отправим уведомление спортсмену
		h.Bot.Send(tgbotapi.NewMessage(fromID, fmt.Sprintf("🚫 Ваш запрос #%d на баллы был отклонён тренером.", id)))

		// Подтверждение тренеру
		h.Bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🚫 Запрос #%d отклонён и удалён.", id)))

	case strings.HasPrefix(text, "/history"):
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Пользователь не найден."))
			return
		}

		args := strings.Fields(text)

		var targetID int64
		var targetName string

		if len(args) == 1 {
			// спортсмен смотрит свою историю
			if user.Role != domain.RoleAthlete {
				h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Тренеры должны указать @username для просмотра истории спортсмена."))
				return
			}
			targetID = chatID
			targetName = user.Username
		} else if len(args) == 2 && user.Role == domain.RoleCoach {
			username := strings.TrimPrefix(args[1], "@")
			targetUser, err := h.Repo.GetUserByUsername(username)
			if err != nil || targetUser == nil || targetUser.Role != domain.RoleAthlete {
				h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Спортсмен с таким username не найден."))
				return
			}
			targetID = targetUser.ID
			targetName = targetUser.Username
		} else {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❗ Формат: /history или /history @username (для тренера)"))
			return
		}

		history, err := h.Repo.GetUserHistory(targetID)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка при получении истории: "+err.Error()))
			return
		}

		if len(history) == 0 {
			msg := "📭 Нет начислений."
			if user.Role == domain.RoleCoach && targetID != chatID {
				msg = fmt.Sprintf("📭 У пользователя @%s пока нет начислений.", targetName)
			}
			h.Bot.Send(tgbotapi.NewMessage(chatID, msg))
			return
		}

		msg := "📜 История начислений:\n\n"
		for _, entry := range history {
			msg += fmt.Sprintf("• ➕ %d баллов — %s\n", entry.Amount, entry.Reason)
		}

		h.Bot.Send(tgbotapi.NewMessage(chatID, msg))
	
	case text == "/my_score":
		user, _ := h.Repo.GetUserByID(chatID)
		if user == nil || user.Role != domain.RoleAthlete {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "🚫 Команда доступна только спортсменам."))
			return
		}

		score, err := h.Repo.GetUserScore(chatID)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка при получении счёта: "+err.Error()))
			return
		}

		msg := fmt.Sprintf("🏅 Твой текущий счёт: %d баллов", score)
		h.Bot.Send(tgbotapi.NewMessage(chatID, msg))

	default:
		h.Bot.Send(tgbotapi.NewMessage(chatID, "❓ Неизвестная команда. Напиши /start."))
	}
}
