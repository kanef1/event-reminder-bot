package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"
	mybot "github.com/kanef1/event-reminder-bot/bot"
	"github.com/kanef1/event-reminder-bot/reminder"
	"github.com/kanef1/event-reminder-bot/storage"
)

func main() {
	if err := godotenv.Load(); 
	err != nil {
        log.Fatal("Не удалось загрузить .env файл")
    }

    token := os.Getenv("TELEGRAM_TOKEN")
    if token == "" {
        log.Fatal("TELEGRAM_TOKEN не задан")
    }

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	b, err := bot.New(token, bot.WithDefaultHandler(mybot.DefaultHandler))
	if err != nil {
		panic(err)
	}

	if err := storage.CleanupPastEvents(); err != nil {
        log.Printf("Ошибка очистки событий: %v", err)
    }

	restoreReminders(ctx, b)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, mybot.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/add", bot.MatchTypePrefix, mybot.AddHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/list", bot.MatchTypeExact, mybot.ListHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, mybot.HelpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/delete", bot.MatchTypePrefix, mybot.DeleteHandler)

	b.Start(ctx)
}


func restoreReminders(ctx context.Context, b *bot.Bot) {
    events, err := storage.LoadEvents()
    if err != nil {
        log.Printf("Ошибка восстановления напоминаний: %v", err)
        return
    }

    for _, e := range events {
        if e.DateTime.After(time.Now()) {
            reminder.ScheduleReminder(ctx, b, e.ChatID, e)
            log.Printf("Восстановлено напоминание: OriginalID=%d", e.OriginalID)
        }
    }
}