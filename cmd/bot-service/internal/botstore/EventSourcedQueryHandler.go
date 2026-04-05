package botstore

import (
	"context"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ QueryHandler = (*EventSourcedQueryHandler)(nil)

type EventSourcedQueryHandler struct {
	log     eventsource.Log
	cursor  int64
	botByID map[string]*Bot
}

type NewEventSourcedQueryHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedQueryHandler(input NewEventSourcedQueryHandlerInput) *EventSourcedQueryHandler {
	return &EventSourcedQueryHandler{
		log:     input.Log,
		botByID: make(map[string]*Bot),
	}
}

func (store *EventSourcedQueryHandler) Get(ctx context.Context, botID string) (bot *Bot, err error) {
	store.catchUp(ctx)
	bot, ok := store.botByID[botID]
	if !ok {
		err = ErrBotNotFound
		return
	}
	userID := contextx.GetUserID(ctx)
	if bot.UserID != userID {
		err = ErrBotForbidden
		return
	}
	return
}

func (store *EventSourcedQueryHandler) List(ctx context.Context) ([]*Bot, error) {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	result := make([]*Bot, 0)
	for _, bot := range store.botByID {
		if bot.UserID != userID {
			continue
		}
		result = append(result, bot)
	}
	return result, nil
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

func (store *EventSourcedQueryHandler) apply(ctx context.Context, event *eventsource.Event) (err error) {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeBotCreated:
		return store.applyBotCreatedEvent(ctx, frame.BotCreatedEvent)
	case EventTypeBotStatusUpdated:
		return store.applyBotStatusUpdatedEvent(ctx, frame.BotStatusUpdatedEvent)
	case EventTypeBotStatusDeleted:
		return store.applyBotStatusDeletedEvent(ctx, frame.BotStatusDeletedEvent)
	}
	return
}

func (store *EventSourcedQueryHandler) applyBotCreatedEvent(ctx context.Context, event *BotCreatedEvent) (err error) {
	store.botByID[event.BotID] = &Bot{
		ID:                event.BotID,
		UserID:            event.UserID,
		AccountID:         event.AccountID,
		BrokerAccountID:   event.BrokerAccountID,
		BrokerType:        event.BrokerType,
		Symbol:            event.Symbol,
		AllocationPercent: event.AllocationPercent,
		ScalpingParams:    event.ScalpingParams,
		Status:            event.Status,
		CreatedAt:         event.CreatedAt,
	}
	return
}

func (store *EventSourcedQueryHandler) applyBotStatusUpdatedEvent(ctx context.Context, event *BotStatusUpdatedEvent) (err error) {
	storeBot, ok := store.botByID[event.BotID]
	if !ok {
		return
	}
	storeBot.Status = event.Status
	return
}

func (store *EventSourcedQueryHandler) applyBotStatusDeletedEvent(ctx context.Context, event *BotStatusDeletedEvent) (err error) {
	delete(store.botByID, event.BotID)
	return
}
