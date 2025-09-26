package db

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/kanef1/event-reminder-bot/model"
)

type Event struct {
	tableName struct{} `pg:"events,discard_unknown_columns"`

	EventId   int       `pg:"eventid"`
	UserTgId  int64     `pg:"usertgid"`
	Message   string    `pg:"message"`
	SendAt    time.Time `pg:"sendat"`
	CreatedAt time.Time `pg:"createdat"`
}

func (e *Event) ToModel() model.Event {
	return model.Event{
		ID:        e.EventId,
		UserTgId:  e.UserTgId,
		Message:   e.Message,
		SendAt:    e.SendAt,
		CreatedAt: e.CreatedAt,
	}
}

func FromModel(m model.Event) Event {
	return Event{
		EventId:   m.ID,
		UserTgId:  m.UserTgId,
		Message:   m.Message,
		SendAt:    m.SendAt,
		CreatedAt: m.CreatedAt,
	}
}

func (db *DB) AddEvent(ctx context.Context, e *Event) error {
	_, err := db.ModelContext(ctx, e).Insert()
	return err
}

func (db *DB) DeleteEvent(ctx context.Context, eventId int) error {
	_, err := db.ModelContext(ctx, &Event{}).
		Where("eventId = ?", eventId).
		Delete()
	return err
}

func (db *DB) GetEvent(ctx context.Context, eventId int) (*Event, error) {
	e := &Event{}
	err := db.ModelContext(ctx, e).
		Where("eventId = ?", eventId).
		Select()
	if err == pg.ErrNoRows {
		return nil, nil
	}
	return e, err
}

func (db *DB) ListEvents(ctx context.Context) ([]Event, error) {
	var events []Event
	err := db.ModelContext(ctx, &events).
		Order("sendAt ASC").
		Select()
	return events, err
}

func (db *DB) ListUserEvents(ctx context.Context, userTgId int64) ([]Event, error) {
	var events []Event
	err := db.ModelContext(ctx, &events).
		Where("userTgId = ?", userTgId).
		Order("sendAt ASC").
		Select()
	return events, err
}

func (db *DB) CleanupPastEvents(ctx context.Context) error {
	_, err := db.ModelContext(ctx, &Event{}).
		Where("sendAt < ?", time.Now()).
		Delete()
	return err
}
