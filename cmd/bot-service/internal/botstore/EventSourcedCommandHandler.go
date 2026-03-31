package botstore

import (
	"context"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ CommandHandler = (*EventSourcedCommandHandler)(nil)

type EventSourcedCommandHandler struct {
	log     eventsource.Log
	cursor  int64
	botByID map[string]*Bot
}

type NewEventSourcedCommandHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedCommandHandler(input NewEventSourcedCommandHandlerInput) *EventSourcedCommandHandler {
	return &EventSourcedCommandHandler{
		log:     input.Log,
		botByID: make(map[string]*Bot),
	}
}

func (store *EventSourcedCommandHandler) Create(ctx context.Context, bot *Bot) (err error) {
	store.catchUp(ctx)
	if _, exists := store.botByID[bot.ID]; exists {
		err = ErrBotAlreadyExists
		return
	}
	fatal.Unless(bot.Status == BotStatusStopped, "new bot must have status stopped")
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeBotCreated),
		BotCreatedEvent: &BotCreatedEvent{
			BotID:             bot.ID,
			UserID:            bot.UserID,
			AccountID:         bot.AccountID,
			BrokerAccountID:   bot.BrokerAccountID,
			BrokerType:        bot.BrokerType,
			Symbol:            bot.Symbol,
			StrategyTradeType: bot.StrategyTradeType,
			AllocationPercent: bot.AllocationPercent,
			Status:            bot.Status,
			CreatedAt:         bot.CreatedAt,
		},
	})
	_, err = store.log.Append(payload)
	return
}

func (store *EventSourcedCommandHandler) UpdateBotStatus(ctx context.Context, botID string, status BotStatus) (err error) {
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

func (store *EventSourcedCommandHandler) Delete(ctx context.Context, botID string) (err error) {
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

func (store *EventSourcedCommandHandler) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedCommandHandler) apply(ctx context.Context, event *eventsource.Event) (err error) {
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

func (store *EventSourcedCommandHandler) applyBotCreatedEvent(ctx context.Context, event *BotCreatedEvent) (err error) {
	store.botByID[event.BotID] = &Bot{
		ID:                event.BotID,
		UserID:            event.UserID,
		AccountID:         event.AccountID,
		BrokerAccountID:   event.BrokerAccountID,
		BrokerType:        event.BrokerType,
		Symbol:            event.Symbol,
		StrategyTradeType: event.StrategyTradeType,
		AllocationPercent: event.AllocationPercent,
		Status:            event.Status,
		CreatedAt:         event.CreatedAt,
	}
	return
}

func (store *EventSourcedCommandHandler) applyBotStatusUpdatedEvent(ctx context.Context, event *BotStatusUpdatedEvent) (err error) {
	storeBot, ok := store.botByID[event.BotID]
	if !ok {
		return
	}
	storeBot.Status = event.Status
	return
}

func (store *EventSourcedCommandHandler) applyBotStatusDeletedEvent(ctx context.Context, event *BotStatusDeletedEvent) (err error) {
	delete(store.botByID, event.BotID)
	return
}
