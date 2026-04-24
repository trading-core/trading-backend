package entrystore

import (
	"context"
	"sort"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ QueryHandler = (*EventSourcedQueryHandler)(nil)

type EventSourcedQueryHandler struct {
	log        eventsource.Log
	cursor     int64
	entryByKey map[string]*Entry
}

type NewEventSourcedQueryHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedQueryHandler(input NewEventSourcedQueryHandlerInput) *EventSourcedQueryHandler {
	return &EventSourcedQueryHandler{
		log:        input.Log,
		entryByKey: make(map[string]*Entry),
	}
}

func (store *EventSourcedQueryHandler) Get(ctx context.Context, date string) (entry *Entry, err error) {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	entry, ok := store.entryByKey[entryKey(userID, date)]
	if !ok {
		err = ErrEntryNotFound
		return
	}
	if entry.UserID != userID {
		err = ErrEntryForbidden
		entry = nil
		return
	}
	return
}

func (store *EventSourcedQueryHandler) List(ctx context.Context, input ListInput) (*ListResult, error) {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	matching := make([]*Entry, 0)
	for _, entry := range store.entryByKey {
		if entry.UserID != userID {
			continue
		}
		if input.From != "" && entry.Date < input.From {
			continue
		}
		if input.To != "" && entry.Date > input.To {
			continue
		}
		matching = append(matching, entry)
	}
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].Date > matching[j].Date
	})
	totalCount := len(matching)
	totalPages := totalCount / input.PageSize
	if totalCount%input.PageSize != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}
	start := input.Page * input.PageSize
	if start > totalCount {
		start = totalCount
	}
	end := start + input.PageSize
	if end > totalCount {
		end = totalCount
	}
	return &ListResult{
		Entries:    matching[start:end],
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func (store *EventSourcedQueryHandler) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedQueryHandler) apply(ctx context.Context, event *eventsource.Event) error {
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

func (store *EventSourcedQueryHandler) applyUpserted(event *EntryUpsertedEvent) error {
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

func (store *EventSourcedQueryHandler) applyDeleted(event *EntryDeletedEvent) error {
	delete(store.entryByKey, entryKey(event.UserID, event.Date))
	return nil
}
