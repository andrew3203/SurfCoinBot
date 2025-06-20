package main

import (
	"log"
	"os"
	"surf_bot/internal/app"
	"surf_bot/internal/handler"
	"surf_bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Initialize DB connection using app-level InitDB
	db := app.InitDB()
	repo := repository.NewUserRepository(db)

	// Setup bot
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	secret := os.Getenv("COACH_SECRET")
	handler := handler.NewTelegramHandler(repo, bot, secret)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Main loop
	for update := range updates {
		handler.HandleUpdate(update)
	}
}
