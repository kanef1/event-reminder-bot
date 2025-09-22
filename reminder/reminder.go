package reminder

import (
	"context"
	"log"
	"time"

	"github.com/go-telegram/bot"
	"github.com/kanef1/event-reminder-bot/db"
	"github.com/kanef1/event-reminder-bot/model"
)

var (
	activeReminders = make(map[int]context.CancelFunc)
)

func ScheduleReminder(ctx context.Context, b *bot.Bot, database *db.DB, chatID int64, event model.Event) context.CancelFunc {
	duration := time.Until(event.SendAt)
	if duration <= 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	RegisterReminder(event.ID, cancel)

	go func() {
		defer cancel()

		select {
		case <-time.After(duration):
			// Проверяем, существует ли событие в БД
			dbEvent, err := database.GetEvent(ctx, event.ID)
			if err != nil {
				log.Printf("Ошибка проверки события: %v", err)
				return
			}

			if dbEvent != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "🔔 Напоминание: " + event.Message,
				})
				log.Printf("Отправлено напоминание: ID=%d", event.ID)
			} else {
				log.Printf("Событие ID=%d было удалено", event.ID)
			}

		case <-ctx.Done():
			log.Printf("Напоминание ID=%d отменено", event.ID)
			return
		}
	}()

	return cancel
}

func RegisterReminder(eventId int, cancel context.CancelFunc) {
	activeReminders[eventId] = cancel
}

func CancelReminder(eventId int) {
	if cancel, exists := activeReminders[eventId]; exists {
		cancel()
		delete(activeReminders, eventId)
		log.Printf("Напоминание ID=%d отменено", eventId)
	}
}
