// internal/util/bot.go
package util

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SafeSend отправляет сообщение и логирует ошибку, если она произошла
func SafeSend(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) {
	if _, err := bot.Send(msg); err != nil {
		log.Printf("⚠️  failed to send message: %v", err)
	}
}
