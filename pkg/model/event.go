package model

import "time"

type Event struct {
	ID         int
	OriginalID int
	ChatID     int64
	Text       string
	DateTime   time.Time
}
