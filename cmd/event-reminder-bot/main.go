package main

import (
	"log"
	"os"

	"github.com/kanef1/event-reminder-bot/pkg/app"

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

	DB_HOST := os.Getenv("DB_HOST")
	DB_PORT := os.Getenv("DB_PORT")
	DB_USER := os.Getenv("DB_USER")
	DB_PASSWORD := os.Getenv("DB_PASSWORD")
	DB_NAME := os.Getenv("DB_NAME")

	if DB_HOST == "" || DB_PORT == "" || DB_USER == "" || DB_PASSWORD == "" || DB_NAME == "" {
		log.Fatal("Не заданы параметры подключения к PostgreSQL")
	}

	a := app.New(token, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
	defer a.Close()

	a.Run()
}
