package main

import (
	"context"
	"log"
	"os"
	"strconv"
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
	LastStartTime time.Time
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

			// 🧾 /rules командасы
			if text == "/rules" {
				msg := "📜 Ойын ережелері:\n\n" +
					"1. Бір адам \"Баста\" немесе /start арқылы ойынды бастайды — ол жасырушы болады.\n" +
					"2. \"Сөзді көру\" батырмасы арқылы жасырушыға сөз шығады (тек оған).\n" +
					"3. Қалғандары сөзді табуға тырысады — дұрыс жауап жазған адам жеңімпаз болады.\n" +
					"4. Жеңімпазда келесі раундта жасырушы болуға 5 секундтық артықшылығы бар.\n" +
					"5. Тек жасырушы \"Сөзді көру\" және \"Келесі сөз\" батырмаларын баса алады.\n" +
					"6. Жаңа раунд бастау үшін кем дегенде 3 минут күту керек."

				bot.SendMessage(ctx, tu.Message(tu.ID(chatID), msg))
				continue
			}
			// 📥 Ответ в личке
			if chat.Type == "private" {
				button := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("🤝 Ботты чатқа қос, балапан").
							WithURL("https://t.me/qoltyrauyn_go_bot?startgroup=true"),
					),
				)
				messageText := "Бұл ойын тек топта ойналады. Мені топқа қосыңыз👇\n\nҰсыныстар бойынша @zhanasbay"
				bot.SendMessage(ctx,
					tu.Message(tu.ID(chatID), messageText).
						WithReplyMarkup(button),
				)
				continue
			}

			// 🚀 /start немесе Баста
			if text == "/start" || text == "Баста" {
				if time.Since(game.LastStartTime) < 3*time.Minute {
					bot.SendMessage(ctx,
						tu.Message(tu.ID(chatID), "❗ Балапан, жасырылған сөзге кем дегенде 3 минут беріледі, көтеніңді қыса ғой, балапан"),
					)
					continue
				}

				game.LastStartTime = time.Now()
				game.HostID = update.Message.From.ID

				hostName := "[" + update.Message.From.FirstName + "](tg://user?id=" + strconv.FormatInt(update.Message.From.ID, 10) + ")"
				msg := "👋 Cәлем, балапан! Обед іштің бе? Кел, ойнайық!\n" +
					"Сөз жасыратын " + hostName

				keyboard := tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("Сөзді көру").WithCallbackData("see_word"),
						tu.InlineKeyboardButton("Келесі сөз").WithCallbackData("next_word"),
					),
				)

				bot.SendMessage(ctx,
					tu.Message(tu.ID(chatID), msg).
						WithParseMode(telego.ModeMarkdown).
						WithReplyMarkup(keyboard),
				)
				continue
			}

			// ✅ Проверка правильного ответа (гибкая)
			if game.CurrentWord != "" && strings.Contains(strings.ToLower(text), strings.ToLower(game.CurrentWord)) {
				sender := update.Message.From

				if sender.ID == game.HostID {
					continue // ❌ ведущий не может угадывать
				}

				game.CurrentWord = ""
				game.WordGuesserID = sender.ID
				game.WordTime = time.Now()
				game.HostID = sender.ID

				winnerLink := "[" + sender.FirstName + "](tg://user?id=" + strconv.FormatInt(sender.ID, 10) + ")"
				msg := "🎉 Жеңімпаз: " + winnerLink + "\nДұрыс жауап: *" + text + "*"

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

				hostLink := "[" + query.From.FirstName + "](tg://user?id=" + strconv.FormatInt(query.From.ID, 10) + ")"
				text := "🎮 Келесі раунд басталды! Келесі сөзді " + hostLink + " жасырады"

				bot.SendMessage(ctx,
					tu.Message(tu.ID(chatID), text).
						WithParseMode(telego.ModeMarkdown).
						WithReplyMarkup(keyboard),
				)
			}

			_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
			})
		}
	}
}
