package bot

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/kanef1/event-reminder-bot/pkg/model"
	"github.com/kanef1/event-reminder-bot/pkg/storage"
)

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

func DeleteHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	events, err := storage.LoadEvents()
	if err != nil {
		log.Printf("Ошибка загрузки событий: %v", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка при загрузке событий",
		})
		return
	}

	for _, e := range events {
		if e.ID == id {
			storage.CancelReminder(e.OriginalID)
			break
		}
	}

	newEvents := make([]model.Event, 0)
	deleted := false
	for _, e := range events {
		if e.ID != id {
			newEvents = append(newEvents, e)
		} else {
			deleted = true
		}
	}

	if !deleted {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🔍 Событие с таким ID не найдено",
		})
		return
	}

	if err := storage.SaveEvents(storage.ReindexEvents(newEvents)); err != nil {
		log.Printf("Ошибка сохранения событий: %v", err)
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

func ListHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	events, err := storage.LoadEvents()
	if err != nil {
		log.Printf("Ошибка загрузки событий: %v", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка при загрузке событий",
		})
		return
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].DateTime.Before(events[j].DateTime)
	})

	if len(events) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🔍 Нет событий",
		})
		return
	}

	var msg strings.Builder
	msg.WriteString("📅 Список событий (от ближайших):\n\n")
	for i, e := range events {
		msg.WriteString(fmt.Sprintf(
			"%d. %s — %s\n",
			i+1,
			e.Text,
			e.DateTime.Format("2006-01-02 15:04"),
		))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg.String(),
	})
}

type BotManager struct {
	b *bot.Bot
}

func NewBotManager(b *bot.Bot) *BotManager {
	return &BotManager{b: b}
}

func (bm BotManager) SendReminder(ctx context.Context, chatID int64, text string) {
	bm.b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "🔔 Напоминание: " + text,
	})
}

func (bm BotManager) AddEvent(ctx context.Context, chatId int64, parts []string) (*model.Event, error) {

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
		return nil, fmt.Errorf("invalid_format")
	}

	if dt.Before(time.Now()) {
		return nil, fmt.Errorf("past_date")
	}

	events, err := storage.LoadEvents()
	if err != nil {
		return nil, err
	}

	event := model.Event{
		OriginalID: int(time.Now().UnixNano()),
		ChatID:     chatId,
		Text:       text,
		DateTime:   dt,
	}

	events = append(events, event)

	events = storage.ReindexEvents(events)

	if err := storage.SaveEvents(events); err != nil {
		log.Printf("Ошибка сохранения событий: %v", err)
		return nil, err
	}

	return &event, nil
}
