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
	"github.com/kanef1/event-reminder-bot/pkg/db"
	"github.com/kanef1/event-reminder-bot/pkg/model"
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

func DeleteHandler(ctx context.Context, b *bot.Bot, update *models.Update, bm *BotManager) {
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

	err = bm.DeleteEventByID(ctx, id)
	if err != nil {
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

func ListHandler(ctx context.Context, b *bot.Bot, update *models.Update, bm *BotManager) {
	events, err := bm.GetUserEvents(ctx, update.Message.Chat.ID)
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
	for i, e := range events {
		msg.WriteString(fmt.Sprintf(
			"%d. %s — %s (ID: %d)\n",
			i+1,
			e.Text,
			e.DateTime.Format("2006-01-02 15:04"),
			e.ID,
		))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg.String(),
	})
}

type BotManager struct {
	b          *bot.Bot
	eventsRepo db.EventsRepo
}

func NewBotManager(b *bot.Bot, eventsRepo db.EventsRepo) *BotManager {
	return &BotManager{b: b, eventsRepo: eventsRepo}
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

	event := &db.Event{
		UserTgID: chatId,
		Message:  text,
		SendAt:   dt,
	}

	addedEvent, err := bm.eventsRepo.AddEvent(ctx, event)
	if err != nil {
		log.Printf("Ошибка сохранения события: %v", err)
		return nil, err
	}

	return &model.Event{
		ID:         addedEvent.ID,
		OriginalID: addedEvent.ID,
		ChatID:     addedEvent.UserTgID,
		Text:       addedEvent.Message,
		DateTime:   addedEvent.SendAt,
	}, nil
}

func (bm BotManager) DeleteEventByID(ctx context.Context, id int) error {
	// Получаем событие из базы данных
	event, err := bm.eventsRepo.EventByID(ctx, id)
	if err != nil {
		return err
	}

	if event == nil {
		return fmt.Errorf("event not found")
	}

	// Удаляем событие из базы данных
	deleted, err := bm.eventsRepo.DeleteEvent(ctx, id)
	if err != nil {
		return err
	}

	if !deleted {
		return fmt.Errorf("event not found")
	}

	return nil
}

func (bm BotManager) GetUserEvents(ctx context.Context, chatID int64) ([]model.Event, error) {
	search := &db.EventSearch{UserTgID: &chatID}
	dbEvents, err := bm.eventsRepo.EventsByFilters(ctx, search, db.PagerNoLimit)
	if err != nil {
		return nil, err
	}

	sort.Slice(dbEvents, func(i, j int) bool {
		return dbEvents[i].SendAt.Before(dbEvents[j].SendAt)
	})

	events := make([]model.Event, len(dbEvents))
	for i, dbEvent := range dbEvents {
		events[i] = model.Event{
			ID:         dbEvent.ID,
			OriginalID: dbEvent.ID,
			ChatID:     dbEvent.UserTgID,
			Text:       dbEvent.Message,
			DateTime:   dbEvent.SendAt,
		}
	}

	return events, nil
}

func (bm BotManager) GetEventByID(ctx context.Context, id int) (*model.Event, error) {
	dbEvent, err := bm.eventsRepo.EventByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dbEvent == nil {
		return nil, nil
	}

	return &model.Event{
		ID:         dbEvent.ID,
		OriginalID: dbEvent.ID,
		ChatID:     dbEvent.UserTgID,
		Text:       dbEvent.Message,
		DateTime:   dbEvent.SendAt,
	}, nil
}
