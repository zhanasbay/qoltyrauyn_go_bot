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

// 👥 Структура состояния игры на каждый чат
type GameState struct {
	CurrentWord   string
	HostID        int64
	WordGuesserID int64
	WordTime      time.Time
}

// 🌍 Глобальная карта: ключ — chatID, значение — GameState
var games = make(map[int64]*GameState)

// 🎯 Получаем или создаём состояние игры для текущего чата
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

			// 🔒 Если пишут в личку — предлагаем добавить бота в чат
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

			// 🚀 Команда /start или "Баста" — начинаем игру
			if text == "/start" || text == "Баста" {
				keyboard := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("Сөзді көру").WithCallbackData("see_word"),
						tu.InlineKeyboardButton("Келесі сөз").WithCallbackData("next_word"),
					),
				)
				bot.SendMessage(ctx,
					tu.Message(tu.ID(chatID), "👋 Cәлем, балапан! Обед іштің бе? Кел, ойнайық").
						WithReplyMarkup(keyboard),
				)
				continue
			}

			// ✅ Проверка правильного ответа
			if game.CurrentWord != "" && strings.EqualFold(text, game.CurrentWord) {
				sender := update.Message.From

				// ❌ Ведущий не может быть победителем
				if sender.ID == game.HostID {
					continue
				}

				game.CurrentWord = ""
				game.WordGuesserID = sender.ID
				game.WordTime = time.Now()
				game.HostID = sender.ID // Победитель — новый ведущий

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

		// 🎮 Обработка нажатий на кнопки
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
					bot.SendMessage(ctx, tu.Message(ctxUser, "⛔ Тек жасырушы ғана бұл батырманы баса алады!"))
					break
				}

				game.CurrentWord = getRandomWord(words)
				game.WordGuesserID = 0
				game.WordTime = time.Now()

				_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
					CallbackQueryID: query.ID,
					Text:            "🔐 Сенің сөзің: " + game.CurrentWord,
					ShowAlert:       true,
				})

			case "next_word":
				if userID != game.HostID {
					bot.SendMessage(ctx, tu.Message(ctxUser, "⛔ Тек жасырушы ғана бұл батырманы баса алады!"))
					break
				}

				game.CurrentWord = getRandomWord(words)
				game.WordTime = time.Now()

				_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
					CallbackQueryID: query.ID,
					Text:            "🔁 Келесі сөз дайын: " + game.CurrentWord,
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

			// ✅ Подтверждение нажатия
			_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
			})
		}
	}
}
