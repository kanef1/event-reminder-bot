package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-telegram/bot"
	botManager "github.com/kanef1/event-reminder-bot/pkg/bot"
	"github.com/kanef1/event-reminder-bot/pkg/botService"
	"github.com/kanef1/event-reminder-bot/pkg/db"
	"github.com/kanef1/event-reminder-bot/pkg/reminder"
)

type App struct {
	b          *bot.Bot
	bm         *botManager.BotManager
	rm         *reminder.ReminderManager
	bs         *botService.BotService
	db         *pg.DB
	eventsRepo db.EventsRepo
}

func New(token, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME string) App {
	var a App

	a.db = pg.Connect(&pg.Options{
		Addr:     DB_HOST + ":" + DB_PORT,
		User:     DB_USER,
		Password: DB_PASSWORD,
		Database: DB_NAME,
	})

	database := db.New(a.db)
	sqlLogger := log.New(os.Stdout, "Q", log.LstdFlags)
	database.AddQueryHook(db.NewQueryLogger(sqlLogger))

	v, err := database.Version()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	log.Println("Postgres version:", v)

	a.eventsRepo = db.NewEventsRepo(a.db)

	b, err := bot.New(token, bot.WithDefaultHandler(botManager.DefaultHandler))
	if err != nil {
		panic(err)
	}
	a.b = b
	a.bm = botManager.NewBotManager(a.b, a.eventsRepo)
	a.rm = reminder.NewReminderManager(a.bm, a.eventsRepo)
	a.bs = botService.NewBotService(b, a.bm, a.rm)

	return a
}

func (a App) Close() {
	if a.db != nil {
		a.db.Close()
	}
}

func (a App) Run() error {
	a.bs.RegisterHandlers()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := a.cleanupPastEvents(); err != nil {
		log.Printf("Ошибка очистки событий: %v", err)
	}

	a.restoreReminders(ctx)

	a.b.Start(ctx)
	return nil
}

func (a App) cleanupPastEvents() error {
	_, err := a.db.ExecContext(context.Background(),
		"DELETE FROM events WHERE \"sendAt\" < NOW()")
	return err
}

func (a App) restoreReminders(ctx context.Context) {
	events, err := a.eventsRepo.EventsByFilters(ctx, &db.EventSearch{}, db.PagerNoLimit)
	if err != nil {
		log.Printf("Ошибка восстановления напоминаний: %v", err)
		return
	}

	for _, e := range events {
		if e.SendAt.After(time.Now()) {
			event := reminder.Event{
				ID:         e.ID,
				OriginalID: e.ID,
				ChatID:     e.UserTgID,
				Text:       e.Message,
				DateTime:   e.SendAt,
			}
			a.rm.ScheduleReminder(ctx, event)
			log.Printf("Восстановлено напоминание: ID=%d", e.ID)
		}
	}
}
