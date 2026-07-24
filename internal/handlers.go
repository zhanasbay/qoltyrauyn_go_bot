package internal

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func HandleUpdates(bot *telego.Bot, ctx context.Context, words []string) {
	updates, err := bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{})
	if err != nil {
		panic("❌ Failed to get updates: " + err.Error())
	}

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, ctx, update)
		}
		if update.CallbackQuery != nil {
			handleCallback(bot, ctx, update.CallbackQuery, words)
		}
	}
}

// ✉️ Обработка текстовых сообщений
func handleMessage(bot *telego.Bot, ctx context.Context, update telego.Update) {
	text := update.Message.Text
	cmd := strings.SplitN(text, "@", 2)[0]

	chat := update.Message.Chat
	chatID := chat.ID
	game := GetGame(chatID)

	switch cmd {
	case "/rules":
		msg := "📜 *Ойын ережелері:*\n\n" +
			"- Ойын барысында бір адам сөз жасырады, қалғандары оны табуға тырысады.\n" +
			"- Жасырушы түбірлес, аударма сөздерді қолданбауы керек.\n" +
			"- Кім бірінші дұрыс жауап берсе — жеңімпаз атанады. Жаңа раунд басталады.\n\n" +
			"📌 *Қосымша ережелер:*\n" +
			"- 'Баста' немесе /start арқылы ойын басталады.\n" +
			"- Тек жасырушы \"Сөзді көру\" және \"Келесі сөз\" батырмаларын баса алады.\n" +
			"- Жеңімпазда келесі раундта жасырушы болуға 5 секундтық артықшылығы бар.\n" +
			"- Бір раундқа (жауабы табылмаған раундқа) кемі 3 минут беріледі — осы уақыт ішінде басқа ойыншы жаңа раунд бастай алмайды."
		bot.SendMessage(ctx, tu.Message(tu.ID(chatID), msg).WithParseMode(telego.ModeMarkdown))
		return

	case "/top":
		top := statStore.GetTopPlayers(chatID, 10)
		if len(top) == 0 {
			bot.SendMessage(ctx, tu.Message(tu.ID(chatID), "Бұл чатта әлі жеңімпаздар жоқ."))
			return
		}
		msg := "🏆 *ТОП-жеңімпаздар (осы чат):*\n\n"
		for i, player := range top {
			line := strconv.Itoa(i+1) + ". [" + player.Username + "](tg://user?id=" + strconv.FormatInt(player.UserID, 10) + ") – " + strconv.Itoa(player.ChatWins) + " жеңіс\n"
			msg += line
		}
		bot.SendMessage(ctx, tu.Message(tu.ID(chatID), msg).WithParseMode(telego.ModeMarkdown))
		return

	case "/global":
		top := statStore.GetGlobalTopPlayers(10)
		if len(top) == 0 {
			bot.SendMessage(ctx, tu.Message(tu.ID(chatID), "Жалпы рейтингте ешкім жеңбеген."))
			return
		}
		msg := "🌐 *Жалпы ТОП-жеңімпаздар:*\n\n"
		for i, player := range top {
			line := strconv.Itoa(i+1) + ". [" + player.Username + "](tg://user?id=" + strconv.FormatInt(player.UserID, 10) + ") – " + strconv.Itoa(player.TotalWins) + " жеңіс\n"
			msg += line
		}
		bot.SendMessage(ctx, tu.Message(tu.ID(chatID), msg).WithParseMode(telego.ModeMarkdown))
		return

	case "/me":
		stats := statStore.GetUserStats(chatID, update.Message.From.ID)
		if stats.TotalWins == 0 {
			bot.SendMessage(ctx, tu.Message(tu.ID(chatID), "Сен әлі жеңімпаз болған жоқсың, балапан 🐥"))
			return
		}
		user := update.Message.From
		msg := "📊 *Сенің статистикаң:*\n\n" +
			"👤 [" + user.FirstName + "](tg://user?id=" + strconv.FormatInt(user.ID, 10) + ")\n" +
			"📍 Бұл чатта: " + strconv.Itoa(stats.ChatWins) + " жеңіс\n" +
			"🌐 Барлығы: " + strconv.Itoa(stats.TotalWins) + " жеңіс"
		bot.SendMessage(ctx, tu.Message(tu.ID(chatID), msg).WithParseMode(telego.ModeMarkdown))
		return
	}

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
		return
	}

	if cmd == "/start" || cmd == "Баста" {
		if time.Since(game.LastStartTime) < 3*time.Minute {
			bot.SendMessage(ctx,
				tu.Message(tu.ID(chatID), "❗ Балапан, жасырылған сөзге кем дегенде 3 минут беріледі, көтеніңді қыса ғой, балапан"),
			)
			return
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
		return
	}

	// Проверка правильного ответа
	if game.CurrentWord != "" && strings.Contains(strings.ToLower(text), strings.ToLower(game.CurrentWord)) {
		sender := update.Message.From

		if sender.ID == game.HostID {
			return
		}
		winnerLink := "[" + sender.FirstName + "](tg://user?id=" + strconv.FormatInt(sender.ID, 10) + ")"
		msg := "🎉 Жеңімпаз: " + winnerLink + "\nДұрыс жауап: *" + game.CurrentWord + "*"

		game.CurrentWord = ""
		game.WordGuesserID = sender.ID
		game.WordTime = time.Now()
		game.HostID = sender.ID

		// 🏆 Обновляем статистику
		statStore.AddWin(chatID, sender.ID, sender.Username)

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

// 🔘 Обработка кнопок
func handleCallback(bot *telego.Bot, ctx context.Context, query *telego.CallbackQuery, words []string) {
	data := query.Data
	userID := query.From.ID
	chatID := query.Message.GetChat().ID
	game := GetGame(chatID)

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
			return
		}

		if game.CurrentWord == "" {
			game.CurrentWord = GetRandomWord(words)
			game.WordGuesserID = 0
			game.WordTime = time.Now()
		}

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
			return
		}

		game.CurrentWord = GetRandomWord(words)
		game.WordTime = time.Now()

		_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            game.CurrentWord,
			ShowAlert:       true,
		})

	case "hide_word":
		// ⏱ Если прошло меньше 5 секунд
		if time.Since(game.WordTime) < 5*time.Second {
			if userID != game.WordGuesserID {
				// ❌ Не победитель → запрет
				_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
					CallbackQueryID: query.ID,
					Text:            "⛔ Жеңімпазға 5 секунд артықшылық беріледі!",
					ShowAlert:       true,
				})
				return
			}
			// ✅ Победитель → можно нажимать (не выходим)
		} else {
			// ⏱ Если прошло больше 5 секунд — побеждает тот, кто успел
			game.HostID = userID
		}

		// 👁️ Показываем слово победителю сразу
		_ = bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            game.CurrentWord,
			ShowAlert:       true,
		})

		// ❌ Удаляем кнопку "Мен жасырамын"
		if query.Message.IsAccessible() {
			msg := query.Message.(*telego.Message)
			_, _ = bot.EditMessageReplyMarkup(ctx, &telego.EditMessageReplyMarkupParams{
				ChatID:    tu.ID(chatID),
				MessageID: msg.MessageID,
				ReplyMarkup: &telego.InlineKeyboardMarkup{
					InlineKeyboard: [][]telego.InlineKeyboardButton{},
				},
			})
		}

		// ✅ Новый раунд
		keyboard := tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("Сөзді көру").WithCallbackData("see_word"),
				tu.InlineKeyboardButton("Келесі сөз").WithCallbackData("next_word"),
			),
		)

		hostLink := "[" + query.From.FirstName + "](tg://user?id=" + strconv.FormatInt(userID, 10) + ")"
		text := "🎮 Келесі сөзді " + hostLink + " жасырады"

		bot.SendMessage(ctx,
			tu.Message(tu.ID(chatID), text).
				WithParseMode(telego.ModeMarkdown).
				WithReplyMarkup(keyboard),
		)

		// 🧠 Генерируем новое слово
		game.CurrentWord = GetRandomWord(words)
		game.WordGuesserID = 0
		game.WordTime = time.Now()

	}
}
