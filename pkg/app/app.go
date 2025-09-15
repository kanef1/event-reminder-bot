package app

import (
	"context"
	"github.com/go-telegram/bot"
	botManager "github.com/kanef1/event-reminder-bot/pkg/bot"
	"github.com/kanef1/event-reminder-bot/pkg/botService"
	"github.com/kanef1/event-reminder-bot/pkg/reminder"
	"github.com/kanef1/event-reminder-bot/pkg/storage"
	"log"
	"os"
	"os/signal"
	"time"
)

type App struct {
	b  *bot.Bot
	bm *botManager.BotManager
	rm *reminder.ReminderManager
	bs *botService.BotService
}

func New(token string) App {
	var a App
	b, err := bot.New(token, bot.WithDefaultHandler(botManager.DefaultHandler))
	if err != nil {
		panic(err)
	}
	a.b = b
	a.bm = botManager.NewBotManager(a.b)
	a.rm = reminder.NewReminderManager(a.bm)
	a.bs = botService.NewBotService(b, a.bm, a.rm)
	return a
}

func (a App) Run() error {
	a.bs.RegisterHandlers()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := storage.CleanupPastEvents(); err != nil {
		log.Printf("Ошибка очистки событий: %v", err)
	}

	a.restoreReminders(ctx)

	a.b.Start(ctx)
	return nil
}

func (a App) restoreReminders(ctx context.Context) {
	events, err := storage.LoadEvents()
	if err != nil {
		log.Printf("Ошибка восстановления напоминаний: %v", err)
		return
	}

	for _, e := range events {
		if e.DateTime.After(time.Now()) {
			a.rm.ScheduleReminder(ctx, e)
			log.Printf("Восстановлено напоминание: OriginalID=%d", e.OriginalID)
		}
	}
}
