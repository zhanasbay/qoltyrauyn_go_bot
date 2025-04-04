package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
)

func main() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get token
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set in .env")
	}

	// Init bot
	bot, err := telego.NewBot(token, telego.WithDefaultLogger(true, true))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot started successfully!")
}
