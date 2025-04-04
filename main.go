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

// 👥 Состояние игры для каждого чата
type GameState struct {
	CurrentWord   string
	HostID        int64
	WordGuesserID int64
	WordTime      time.Time
	LastStartTime time.Time // 🕒 время последнего /start
}

// 🌍 Карта: chatID → GameState
var games = make(map[int64]*GameState)

func getGame(chatID int64) *GameState {
	if game, ok := games[chatID]; ok {
		return game
	}
	games[chatID] = &GameState{}
	return games[chatID]
}

func main() {
	_ = godotenv.Load()
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
			chatID := chat.ID
			game := getGame(chatID)

			// 📥 Ответ в личке
			if chat.Type == "private" {
				button := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("🤝 Ботты чатқа қос, балапан").
							WithURL("https://t.me/qoltyrauyn_go_bot?startgroup=true"),
					),
				)
				bot.SendMessage(ctx,
					tu.Message(tu.ID(chatID), "Бұл ойын тек топта ойналады. Мені топқа қосыңыз👇").
						WithReplyMarkup(button),
				)
				continue
			}

			// 🚀 /start немесе Баста
			if text == "/start" || text == "Баста" {
				if time.Since(game.LastStartTime) < 3*time.Minute {
					bot.SendMessage(ctx,
						tu.Message(tu.ID(chatID), "❗ Балапан, жасырылған сөзге минимум 3 минут, көтеніңді қыса ғой, балапан"),
					)
					continue
				}

				game.LastStartTime = time.Now()
				game.HostID = update.Message.From.ID

				hostUsername := update.Message.From.Username
				msg := "👋 Cәлем, балапан! Обед іштің бе? Кел, ойнайық!\n" +
					"Сөз жасыратын @" + hostUsername

				keyboard := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("Сөзді көру").WithCallbackData("see_word"),
						tu.InlineKeyboardButton("Келесі сөз").WithCallbackData("next_word"),
					),
				)

				bot.SendMessage(ctx,
					tu.Message(tu.ID(chatID), msg).
						WithReplyMarkup(keyboard),
				)
				continue
			}

			// ✅ Проверка правильного ответа
			if game.CurrentWord != "" && strings.Contains(strings.ToLower(text), strings.ToLower(game.CurrentWord)) {

				sender := update.Message.From

				if sender.ID == game.HostID {
					continue // ❌ ведущий не может угадывать
				}

				game.CurrentWord = ""
				game.WordGuesserID = sender.ID
				game.WordTime = time.Now()
				game.HostID = sender.ID

				msg := "🎉 Жеңімпаз: @" + sender.Username + "\nДұрыс жауап: *" + text + "*"

				keyboard := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("Мен жасырамын").WithCallbackData("hide_word"),
					),
				)

				bot.SendMessage(ctx,
					tu.Message(tu.ID(chatID), msg).
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
			chatID := query.Message.GetChat().ID
			game := getGame(chatID)

			switch data {
			case "see_word":
				if game.HostID == 0 {
					game.HostID = userID
				}
				if userID != game.HostID {
					_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
						CallbackQueryID: query.ID,
						Text:            "⛔ Тек жасырушы ғана бұл батырманы баса алады!",
						ShowAlert:       true,
					})

					break
				}

				game.CurrentWord = getRandomWord(words)
				game.WordGuesserID = 0
				game.WordTime = time.Now()

				_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
					CallbackQueryID: query.ID,
					Text:            game.CurrentWord,
					ShowAlert:       true,
				})

			case "next_word":
				if userID != game.HostID {
					_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
						CallbackQueryID: query.ID,
						Text:            "⛔ Тек жасырушы ғана бұл батырманы баса алады!",
						ShowAlert:       true,
					})

					break
				}

				game.CurrentWord = getRandomWord(words)
				game.WordTime = time.Now()

				_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
					CallbackQueryID: query.ID,
					Text:            game.CurrentWord,
					ShowAlert:       true,
				})

			case "hide_word":
				if userID != game.WordGuesserID && time.Since(game.WordTime) < 5*time.Second {
					bot.SendMessage(ctx,
						tu.Message(ctxUser, "⛔ Тек жеңімпаз ғана 5 секунд ішінде баса алады!"))
					break
				}

				keyboard := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("Сөзді көру").WithCallbackData("see_word"),
						tu.InlineKeyboardButton("Келесі сөз").WithCallbackData("next_word"),
					),
				)

				hostMention := "@" + query.From.Username
				text := "🎮 Келесі раунд басталды! Келесі сөзді " + hostMention + " жасырады"

				bot.SendMessage(ctx,
					tu.Message(tu.ID(chatID), text).
						WithReplyMarkup(keyboard),
				)
			}

			_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
			})
		}
	}
}
