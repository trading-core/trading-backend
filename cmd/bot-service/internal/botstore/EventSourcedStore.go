package botstore

import (
	"context"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ Store = (*EventSourcedStore)(nil)

type EventSourcedStore struct {
	log     eventsource.Log
	cursor  int64
	botByID map[string]*Bot
}

type NewEventSourcedStoreInput struct {
	Log eventsource.Log
}

func NewEventSourcedStore(input NewEventSourcedStoreInput) *EventSourcedStore {
	return &EventSourcedStore{
		log:     input.Log,
		botByID: make(map[string]*Bot),
	}
}

func (store *EventSourcedStore) Create(ctx context.Context, bot *Bot) (err error) {
	store.catchUp(ctx)
	if _, exists := store.botByID[bot.ID]; exists {
		err = ErrBotAlreadyExists
		return
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeBotCreated),
		BotCreatedEvent: &BotCreatedEvent{
			BotID:           bot.ID,
			UserID:          bot.UserID,
			AccountID:       bot.AccountID,
			BrokerAccountID: bot.BrokerAccountID,
			BrokerType:      bot.BrokerType,
			Symbol:          bot.Symbol,
			StrategyTradeType: bot.StrategyTradeType,
			CreatedAt:       bot.CreatedAt,
		},
	})
	_, err = store.log.Append(payload)
	return
}

func (store *EventSourcedStore) UpdateBotStatus(ctx context.Context, botID string, status BotStatus) (err error) {
	store.catchUp(ctx)
	storeBot, ok := store.botByID[botID]
	if !ok {
		err = ErrBotNotFound
		return
	}
	userID := contextx.GetUserID(ctx)
	if storeBot.UserID != userID {
		return ErrBotForbidden
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeBotStatusUpdated),
		BotStatusUpdatedEvent: &BotStatusUpdatedEvent{
			BotID:  botID,
			Status: status,
		},
	})
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedStore) Get(ctx context.Context, botID string) (bot *Bot, err error) {
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

func (store *EventSourcedStore) List(ctx context.Context) ([]*Bot, error) {
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

func (store *EventSourcedStore) Delete(ctx context.Context, botID string) (err error) {
	store.catchUp(ctx)
	storeBot, ok := store.botByID[botID]
	if !ok {
		err = ErrBotNotFound
		return
	}
	userID := contextx.GetUserID(ctx)
	if storeBot.UserID != userID {
		err = ErrBotForbidden
		return
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeBotStatusDeleted),
		BotStatusDeletedEvent: &BotStatusDeletedEvent{
			BotID: botID,
		},
	})
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedStore) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.CatchUpInput{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedStore) apply(ctx context.Context, event *eventsource.Event) (err error) {
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

func (store *EventSourcedStore) applyBotCreatedEvent(ctx context.Context, event *BotCreatedEvent) (err error) {
	symbol := event.Symbol
	if symbol == "" {
		symbol = event.Name
	}
	store.botByID[event.BotID] = &Bot{
		ID:                event.BotID,
		UserID:            event.UserID,
		AccountID:         event.AccountID,
		BrokerType:        event.BrokerType,
		Symbol:            symbol,
		StrategyTradeType: event.StrategyTradeType,
		Status:            BotStatusStopped,
		CreatedAt:         event.CreatedAt,
	}
	return
}

func (store *EventSourcedStore) applyBotStatusUpdatedEvent(ctx context.Context, event *BotStatusUpdatedEvent) (err error) {
	storeBot, ok := store.botByID[event.BotID]
	if !ok {
		return
	}
	storeBot.Status = event.Status
	return
}

func (store *EventSourcedStore) applyBotStatusDeletedEvent(ctx context.Context, event *BotStatusDeletedEvent) (err error) {
	delete(store.botByID, event.BotID)
	return
}
