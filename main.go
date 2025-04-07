package main

import (
	"context"
	"log"
	"os"

	"qoltyrauyn_go_bot/internal"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
)

func main() {
	// Загружаем переменные окружения
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN is not set")
	}

	// Создаём бота
	bot, err := telego.NewBot(token, telego.WithDefaultLogger(true, true))
	if err != nil {
		log.Fatal("❌ Failed to create bot:", err)
	}

	log.Println("✅ Bot is running...")

	ctx := context.Background()

	// Загружаем слова
	words := internal.LoadWordsFromFile("words.txt")

	// Обработка обновлений
	internal.HandleUpdates(bot, ctx, words)
}
