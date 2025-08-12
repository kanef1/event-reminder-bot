package bot

import (
	"context"
	"fmt"
	"strings"
	"time"
	"log"
	"strconv"
	"sort"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/kanef1/event-reminder-bot/model"
	"github.com/kanef1/event-reminder-bot/reminder"
	"github.com/kanef1/event-reminder-bot/storage"
)

func DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Нет такой команды, используйте /help чтобы посмотреть доступные команды команд",
	})
}

func StartHandler(ctx context.Context, b *bot.Bot, update *models.Update){
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Добрый день, данный бот предназначен для простого планирования.\n" +
		"Список умений:\n" +
		"Добавить событие: /add 2025-08-08 21:05 <Текст>\n" +
		"Список событий: /list \n" +
        "Удалить событие: /delete id\n"+
        "Список команд: /help",
	})
}

func HelpHandler(ctx context.Context, b *bot.Bot, update *models.Update){
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Список умений:\n" +
		"Добавить событие: /add 2025-08-08 21:05 <Текст>\n" +
		"Список событий: /list\n"+
        "Удалить событие: /delete id\n"+
        "Список команд: /help",
	})
}

func AddHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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


	if dt.Before(time.Now()){
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❗ Недопустимый формат даты (событие должно быть в будущем)",
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

     event := model.Event{
        OriginalID: int(time.Now().UnixNano()),
        ChatID:     update.Message.Chat.ID,
        Text:       text,
        DateTime:   dt,
    }

    events = append(events, event)
    
    events = storage.ReindexEvents(events)

    if err := storage.SaveEvents(events); err != nil {
        log.Printf("Ошибка сохранения событий: %v", err)
        b.SendMessage(ctx, &bot.SendMessageParams{
            ChatID: update.Message.Chat.ID,
            Text:   "❌ Ошибка при сохранении события",
        })
        return
    }

    reminder.ScheduleReminder(ctx, b, update.Message.Chat.ID, event)
    
    b.SendMessage(ctx, &bot.SendMessageParams{
        ChatID: update.Message.Chat.ID,
        Text:   "✅ Событие добавлено!",
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


