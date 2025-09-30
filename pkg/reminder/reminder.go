package reminder

import (
	"context"
	"log"
	"sync"
	"time"

	botManager "github.com/kanef1/event-reminder-bot/pkg/bot"
	"github.com/kanef1/event-reminder-bot/pkg/db"
)

type Event struct {
	ID         int
	OriginalID int
	ChatID     int64
	Text       string
	DateTime   time.Time
}

type ReminderManager struct {
	bm         *botManager.BotManager
	eventsRepo db.EventsRepo
	cancels    map[int]context.CancelFunc
	mu         sync.RWMutex
}

func NewReminderManager(bm *botManager.BotManager, eventsRepo db.EventsRepo) *ReminderManager {
	return &ReminderManager{
		bm:         bm,
		eventsRepo: eventsRepo,
		cancels:    make(map[int]context.CancelFunc),
	}
}

func (rm *ReminderManager) ScheduleReminder(ctx context.Context, e Event) context.CancelFunc {
	duration := time.Until(e.DateTime)
	if duration <= 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)

	rm.mu.Lock()
	rm.cancels[e.ID] = cancel
	rm.mu.Unlock()

	go func() {
		defer func() {
			rm.mu.Lock()
			delete(rm.cancels, e.ID)
			rm.mu.Unlock()
			cancel()
		}()

		select {
		case <-time.After(duration):
			event, err := rm.bm.GetEventByID(ctx, e.ID)
			if err != nil {
				log.Printf("Ошибка проверки события: %v", err)
				return
			}

			if event != nil {
				rm.bm.SendReminder(ctx, e.ChatID, e.Text)
				log.Printf("Отправлено напоминание: ID=%d", e.ID)

				err := rm.bm.DeleteEventByID(ctx, e.ID)
				if err != nil {
					log.Printf("Ошибка удаления события после напоминания: %v", err)
				}
			} else {
				log.Printf("Событие ID=%d было удалено", e.ID)
			}

		case <-ctx.Done():
			log.Printf("Напоминание ID=%d отменено", e.ID)
			return
		}
	}()

	return cancel
}

func (rm *ReminderManager) CancelReminder(eventID int) {
	rm.mu.RLock()
	cancel, exists := rm.cancels[eventID]
	rm.mu.RUnlock()

	if exists {
		cancel()

		rm.mu.Lock()
		delete(rm.cancels, eventID)
		rm.mu.Unlock()

		log.Printf("Напоминание ID=%d отменено", eventID)
	}
}
