package entrystore

import (
	"context"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ CommandHandler = (*EventSourcedCommandHandler)(nil)

type EventSourcedCommandHandler struct {
	log       eventsource.Log
	cursor    int64
	entryByKey map[string]*Entry
}

type NewEventSourcedCommandHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedCommandHandler(input NewEventSourcedCommandHandlerInput) *EventSourcedCommandHandler {
	return &EventSourcedCommandHandler{
		log:        input.Log,
		entryByKey: make(map[string]*Entry),
	}
}

func (store *EventSourcedCommandHandler) UpsertEntry(ctx context.Context, entry *Entry) error {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	entry.UserID = userID
	existing, exists := store.entryByKey[entryKey(userID, entry.Date)]
	createdAt := entry.CreatedAt
	if exists {
		createdAt = existing.CreatedAt
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeEntryUpserted),
		EntryUpsertedEvent: &EntryUpsertedEvent{
			UserID:            userID,
			Date:              entry.Date,
			Notes:             entry.Notes,
			Tags:              entry.Tags,
			Mood:              entry.Mood,
			DisciplineScore:   entry.DisciplineScore,
			ScreenshotFileIDs: entry.ScreenshotFileIDs,
			CreatedAt:         createdAt,
			UpdatedAt:         entry.UpdatedAt,
		},
	})
	_, err := store.log.Append(payload)
	return err
}

func (store *EventSourcedCommandHandler) DeleteEntry(ctx context.Context, input DeleteEntryInput) error {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	entry, ok := store.entryByKey[entryKey(userID, input.Date)]
	if !ok {
		return ErrEntryNotFound
	}
	if entry.UserID != userID {
		return ErrEntryForbidden
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeEntryDeleted),
		EntryDeletedEvent: &EntryDeletedEvent{
			UserID:    userID,
			Date:      input.Date,
			UpdatedAt: input.UpdatedAt,
		},
	})
	_, err := store.log.Append(payload)
	return err
}

func (store *EventSourcedCommandHandler) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedCommandHandler) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeEntryUpserted:
		return store.applyUpserted(frame.EntryUpsertedEvent)
	case EventTypeEntryDeleted:
		return store.applyDeleted(frame.EntryDeletedEvent)
	}
	return nil
}

func (store *EventSourcedCommandHandler) applyUpserted(event *EntryUpsertedEvent) error {
	store.entryByKey[entryKey(event.UserID, event.Date)] = &Entry{
		UserID:            event.UserID,
		Date:              event.Date,
		Notes:             event.Notes,
		Tags:              event.Tags,
		Mood:              event.Mood,
		DisciplineScore:   event.DisciplineScore,
		ScreenshotFileIDs: event.ScreenshotFileIDs,
		CreatedAt:         event.CreatedAt,
		UpdatedAt:         event.UpdatedAt,
	}
	return nil
}

func (store *EventSourcedCommandHandler) applyDeleted(event *EntryDeletedEvent) error {
	delete(store.entryByKey, entryKey(event.UserID, event.Date))
	return nil
}

func entryKey(userID string, date string) string {
	return userID + "|" + date
}
