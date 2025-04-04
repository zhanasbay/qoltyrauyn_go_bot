package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// Глобальные переменные
var currentWord string
var wordGuesserID int64
var wordTime time.Time
var hostID int64

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("❌ Failed to load .env file")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN is not set")
	}

	bot, err := telego.NewBot(token, telego.WithDefaultLogger(true, true))
	if err != nil {
		log.Fatal("❌ Failed to create bot:", err)
	}

	ctx := context.Background()
	words := loadWordsFromFile("words.txt")

	updates, err := bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{})
	if err != nil {
		log.Fatal("❌ Failed to get updates:", err)
	}

	log.Println("✅ Bot is running...")

	for update := range updates {
		if update.Message != nil {
			text := update.Message.Text
			chat := update.Message.Chat
			chatID := tu.ID(chat.ID)

			// 💬 Ответ в личке
			if chat.Type == "private" {
				button := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("🤝 Ботты чатқа қос, балапан").
							WithURL("https://t.me/qoltyrauyn_go_bot?startgroup=true"),
					),
				)
				bot.SendMessage(ctx,
					tu.Message(chatID, "Бұл ойын тек топта ойналады. Мені топқа қосыңыз👇").
						WithReplyMarkup(button),
				)
				continue
			}

			// 🚀 Запуск
			if text == "/start" || text == "Баста" {
				keyboard := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("Сөзді көру").WithCallbackData("see_word"),
						tu.InlineKeyboardButton("Келесі сөз").WithCallbackData("next_word"),
					),
				)
				bot.SendMessage(ctx,
					tu.Message(chatID, "👋 Cәлем, балапан! Обед іштің бе? Кел, ойнайық").
						WithReplyMarkup(keyboard),
				)
				continue
			}

			// ✅ Проверка ответа
			if currentWord != "" && strings.EqualFold(text, currentWord) {
				sender := update.Message.From

				// ❌ Ведущий не может быть победителем
				if sender.ID == hostID {
					continue
				}

				currentWord = ""
				wordGuesserID = sender.ID
				wordTime = time.Now()
				hostID = sender.ID

				msg := "🎉 Жеңімпаз: @" + sender.Username + "\nДұрыс жауап: *" + text + "*"

				keyboard := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("Мен жасырамын").WithCallbackData("hide_word"),
					),
				)

				bot.SendMessage(ctx,
					tu.Message(chatID, msg).
						WithParseMode(telego.ModeMarkdown).
						WithReplyMarkup(keyboard),
				)
			}
		}

		if update.CallbackQuery != nil {
			query := update.CallbackQuery
			data := query.Data
			userID := query.From.ID
			ctxUser := tu.ID(userID)

			log.Printf("User @%s clicked: %s", query.From.Username, data)

			switch data {

			case "see_word":
				if hostID == 0 {
					hostID = query.From.ID
				}
				if query.From.ID != hostID {
					bot.SendMessage(ctx, tu.Message(ctxUser, "⛔ Тек жасырушы ғана бұл батырманы баса алады!"))
					break
				}

				currentWord = getRandomWord(words)
				wordGuesserID = 0
				wordTime = time.Now()

				_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
					CallbackQueryID: query.ID,
					Text:            "🔐 Сенің сөзің: " + currentWord,
					ShowAlert:       true,
				})

			case "next_word":
				if query.From.ID != hostID {
					bot.SendMessage(ctx, tu.Message(ctxUser, "⛔ Тек жасырушы ғана бұл батырманы баса алады!"))
					break
				}

				currentWord = getRandomWord(words)
				wordTime = time.Now()

				_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
					CallbackQueryID: query.ID,
					Text:            "🔁 Келесі сөз дайын: " + currentWord,
					ShowAlert:       true,
				})

			case "hide_word":
				if userID != wordGuesserID && time.Since(wordTime) < 5*time.Second {
					bot.SendMessage(ctx,
						tu.Message(ctxUser, "⛔ Тек жеңімпаз ғана 5 секунд ішінде баса алады!"))
				} else {
					keyboard := tu.InlineKeyboard(
						tu.InlineKeyboardRow(
							tu.InlineKeyboardButton("Сөзді көру").WithCallbackData("see_word"),
							tu.InlineKeyboardButton("Келесі сөз").WithCallbackData("next_word"),
						),
					)

					hostMention := "@" + query.From.Username
					text := "🎮 Келесі раунд басталды! Келесі сөзді " + hostMention + " жасырады"

					bot.SendMessage(ctx,
						tu.Message(tu.ID(query.Message.GetChat().ID), text).
							WithReplyMarkup(keyboard),
					)
				}
			}

			_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
			})
		}
	}
}
