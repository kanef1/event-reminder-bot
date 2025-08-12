package reminder

import (
    "context"
    "log"
    "time"

    "github.com/go-telegram/bot"
    "github.com/kanef1/event-reminder-bot/model"
    "github.com/kanef1/event-reminder-bot/storage"
)

func ScheduleReminder(ctx context.Context, b *bot.Bot, chatID int64, e model.Event) context.CancelFunc {
    duration := time.Until(e.DateTime)
    if duration <= 0 {
        return nil
    }

    ctx, cancel := context.WithCancel(ctx)
    storage.RegisterReminder(e.OriginalID, cancel)

    go func() {
        defer cancel()
        
        select {
        case <-time.After(duration):
            events, err := storage.LoadEvents()
            if err != nil {
                log.Printf("Ошибка загрузки событий: %v", err)
                return
            }

            exists := false
            for _, event := range events {
                if event.OriginalID == e.OriginalID {
                    exists = true
                    break
                }
            }

            if exists {
                b.SendMessage(ctx, &bot.SendMessageParams{
                    ChatID: chatID,
                    Text:   "🔔 Напоминание: " + e.Text,
                })
                log.Printf("Отправлено напоминание: OriginalID=%d", e.OriginalID)
            } else {
                log.Printf("Событие OriginalID=%d было удалено", e.OriginalID)
            }

        case <-ctx.Done():
            log.Printf("Напоминание OriginalID=%d отменено", e.OriginalID)
            return
        }
    }()

    return cancel
}


