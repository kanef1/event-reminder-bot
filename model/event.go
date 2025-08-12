package model

import "time"

type Event struct {
    ID         int       `json:"id"`
    OriginalID int       `json:"original_id"` 
    ChatID     int64     `json:"chat_id"`
    Text       string    `json:"text"`
    DateTime   time.Time `json:"datetime"`
}