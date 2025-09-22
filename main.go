package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"

	mybot "github.com/kanef1/event-reminder-bot/bot"
	"github.com/kanef1/event-reminder-bot/db"
	"github.com/kanef1/event-reminder-bot/reminder"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Не удалось загрузить .env файл")
	}

	dbc := pg.Connect(&pg.Options{
		Addr:     os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
	})
	defer dbc.Close()

	database := db.New(dbc)
	sqlLogger := log.New(os.Stdout, "Q", log.LstdFlags)
	database.AddQueryHook(db.NewQueryLogger(sqlLogger))

	v, err := database.Version()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	log.Println("Postgres version:", v)

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN не задан")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := database.CleanupPastEvents(ctx); err != nil {
		log.Printf("Ошибка очистки событий: %v", err)
	}

	b, err := bot.New(token, bot.WithDefaultHandler(mybot.DefaultHandler))
	if err != nil {
		panic(err)
	}

	restoreReminders(ctx, b, &database)

	mybot.RegisterHandlers(b, &database)

	log.Println("Бот запущен...")
	b.Start(ctx)
}

func restoreReminders(ctx context.Context, b *bot.Bot, database *db.DB) {
	events, err := database.ListEvents(ctx)
	if err != nil {
		log.Printf("Ошибка восстановления напоминаний: %v", err)
		return
	}

	for _, e := range events {
		if e.SendAt.After(time.Now()) {
			modelEvent := e.ToModel()
			reminder.ScheduleReminder(ctx, b, database, e.UserTgId, modelEvent)
			log.Printf("Восстановлено напоминание: EventId=%d", e.EventId)
		}
	}
}
