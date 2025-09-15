package main

import (
	"github.com/kanef1/event-reminder-bot/pkg/app"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Не удалось загрузить .env файл")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN не задан")
	}

	a := app.New(token)

	a.Run()
}
