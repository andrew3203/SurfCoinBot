// TelegramHandler encapsulates dependencies for handling updates.
package handler

import (
	repo "surf_bot/internal/repository" // 👈 добавь псевдоним repo

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramHandler struct {
	Repo        *repo.UserRepository
	SecretCoach string
	Bot         *tgbotapi.BotAPI
}

// NewTelegramHandler constructs a new handler instance.
func NewTelegramHandler(r *repo.UserRepository, bot *tgbotapi.BotAPI, secret string) *TelegramHandler {
	return &TelegramHandler{Repo: r, SecretCoach: secret, Bot: bot}
}
