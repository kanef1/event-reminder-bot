package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kanef1/event-reminder-bot/pkg/model"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

var (
	filePath        = "events.json"
	fileMu          sync.Mutex
	remindersMu     sync.Mutex
	activeReminders = make(map[int]context.CancelFunc)
)

func LoadEvents() ([]model.Event, error) {
	fileMu.Lock()
	defer fileMu.Unlock()

	file, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Event{}, nil
		}
		return nil, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	var events []model.Event
	if err := json.Unmarshal(file, &events); err != nil {
		return nil, fmt.Errorf("ошибка разбора JSON: %v", err)
	}
	return events, nil
}

func SaveEvents(events []model.Event) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %v", err)
	}

	return os.WriteFile("events.json", data, 0644)
}

func CleanupPastEvents() error {
	events, err := LoadEvents()
	if err != nil {
		return err
	}

	newEvents := make([]model.Event, 0)
	for _, e := range events {
		if e.DateTime.After(time.Now()) {
			newEvents = append(newEvents, e)
		} else {
			log.Printf("Очистка прошедшего события ID=%d", e.ID)
			CancelReminder(e.OriginalID)
		}
	}

	return SaveEvents(ReindexEvents(newEvents))
}

func ReindexEvents(events []model.Event) []model.Event {
	if len(events) == 0 {
		return events
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].DateTime.Before(events[j].DateTime)
	})

	for i := range events {
		events[i].ID = i + 1
	}
	return events
}

func CancelReminder(originalID int) {
	remindersMu.Lock()
	defer remindersMu.Unlock()

	if cancel, exists := activeReminders[originalID]; exists {
		cancel()
		delete(activeReminders, originalID)
		log.Printf("Напоминание OriginalID=%d отменено", originalID)
	}
}

func RegisterReminder(originalID int, cancel context.CancelFunc) {
	remindersMu.Lock()
	defer remindersMu.Unlock()
	activeReminders[originalID] = cancel
}
