package model

import "time"

type Event struct {
	ID        int       `json:"id"`
	UserTgId  int64     `json:"user_tg_id"`
	Message   string    `json:"message"`
	SendAt    time.Time `json:"send_at"`
	CreatedAt time.Time `json:"created_at"`
}
