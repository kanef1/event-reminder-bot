package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/kanef1/event-reminder-bot/db"
	"github.com/kanef1/event-reminder-bot/reminder"
)

func RegisterHandlers(b *bot.Bot, database *db.DB) {
	b.RegisterHandler(bot.HandlerTypeMessageText, "/add", bot.MatchTypePrefix, AddHandler(database))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/list", bot.MatchTypeExact, ListHandler(database))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/delete", bot.MatchTypeExact, DeleteHandler(database))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, HelpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, StartHandler)
}

func DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Нет такой команды, используйте /help чтобы посмотреть доступные команды команд",
	})
}

func StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: "Добрый день, данный бот предназначен для простого планирования.\n" +
			"Список умений:\n" +
			"Добавить событие: /add 2025-08-08 21:05 <Текст>\n" +
			"Список событий: /list \n" +
			"Удалить событие: /delete id\n" +
			"Список команд: /help",
	})
}

func HelpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: "Список умений:\n" +
			"Добавить событие: /add 2025-08-08 21:05 <Текст>\n" +
			"Список событий: /list\n" +
			"Удалить событие: /delete id\n" +
			"Список команд: /help",
	})
}

func AddHandler(database *db.DB) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		args := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/add"))
		parts := strings.SplitN(args, " ", 3)

		if len(parts) < 3 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❗ Формат: /add 2025-08-06 15:00 Текст",
			})
			return
		}

		datePart := parts[0]
		timePart := parts[1]
		text := parts[2]

		loc, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			log.Println("Ошибка загрузки часового пояса:", err)
			loc = time.Local
		}

		dt, err := time.ParseInLocation("2006-01-02 15:04", datePart+" "+timePart, loc)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❗ Неверный формат времени. Используй: 2025-08-06 15:00",
			})
			return
		}

		if dt.Before(time.Now()) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❗ Недопустимый формат даты (событие должно быть в будущем)",
			})
			return
		}

		event := &db.Event{
			UserTgId:  update.Message.Chat.ID,
			Message:   text,
			SendAt:    dt,
			CreatedAt: time.Now(),
		}

		if err := database.AddEvent(ctx, event); err != nil {
			log.Printf("Ошибка сохранения события: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❌ Ошибка при сохранении события",
			})
			return
		}

		modelEvent := event.ToModel()
		reminder.ScheduleReminder(ctx, b, database, update.Message.Chat.ID, modelEvent)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("✅ Событие добавлено с ID %d!", event.EventId),
		})
	}
}

func DeleteHandler(database *db.DB) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		args := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/delete"))
		if args == "" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❗ Укажите ID события, например: /delete 123",
			})
			return
		}

		id, err := strconv.Atoi(args)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❗ ID должен быть числом",
			})
			return
		}

		reminder.CancelReminder(id)

		if err := database.DeleteEvent(ctx, id); err != nil {
			log.Printf("Ошибка удаления события: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❌ Ошибка при удалении события",
			})
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "✅ Событие удалено!",
		})
	}
}

func ListHandler(database *db.DB) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		events, err := database.ListUserEvents(ctx, update.Message.Chat.ID)
		if err != nil {
			log.Printf("Ошибка загрузки событий: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❌ Ошибка при загрузке событий",
			})
			return
		}

		if len(events) == 0 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "🔍 Нет событий",
			})
			return
		}

		var msg strings.Builder
		msg.WriteString("📅 Список событий (от ближайших):\n\n")
		for _, e := range events {
			msg.WriteString(fmt.Sprintf(
				"%d. %s — %s\n",
				e.EventId,
				e.Message,
				e.SendAt.Format("2006-01-02 15:04"),
			))
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   msg.String(),
		})
	}
}
