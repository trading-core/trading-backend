package entrystore

import "github.com/kduong/trading-backend/internal/eventsource"

const (
	EventTypeEntryUpserted eventsource.EventType = "entry_upserted"
	EventTypeEntryDeleted  eventsource.EventType = "entry_deleted"
)

type EventFrame struct {
	eventsource.EventBase
	EntryUpsertedEvent *EntryUpsertedEvent `json:"entry_upserted_event,omitempty"`
	EntryDeletedEvent  *EntryDeletedEvent  `json:"entry_deleted_event,omitempty"`
}

type EntryUpsertedEvent struct {
	UserID            string   `json:"user_id"`
	Date              string   `json:"date"`
	Notes             string   `json:"notes,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Mood              string   `json:"mood,omitempty"`
	DisciplineScore   int      `json:"discipline_score,omitempty"`
	ScreenshotFileIDs []string `json:"screenshot_file_ids,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

type EntryDeletedEvent struct {
	UserID    string `json:"user_id"`
	Date      string `json:"date"`
	UpdatedAt string `json:"updated_at"`
}
