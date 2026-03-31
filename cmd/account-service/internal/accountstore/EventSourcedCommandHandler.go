package accountstore

import (
	"context"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/logger"
)

var _ CommandHandler = (*EventSourcedCommandHandler)(nil)

type EventSourcedCommandHandler struct {
	log           eventsource.Log
	cursor        int64
	accountByID   map[string]*Account
	tastyTradeIDs map[string]struct{}
}

type NewEventSourcedCommandHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedCommandHandler(input NewEventSourcedCommandHandlerInput) *EventSourcedCommandHandler {
	return &EventSourcedCommandHandler{
		log:           input.Log,
		accountByID:   make(map[string]*Account),
		tastyTradeIDs: make(map[string]struct{}),
	}
}

func (store *EventSourcedCommandHandler) Create(ctx context.Context, input CreateInput) error {
	store.catchUp(ctx)
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeAccountCreated),
		AccountCreatedEvent: &AccountCreatedEvent{
			AccountID:   input.AccountID,
			AccountName: input.AccountName,
			UserID:      contextx.GetUserID(ctx),
		},
	})
	_, err := store.log.Append(payload)
	fatal.OnError(err)
	return nil
}

func (store *EventSourcedCommandHandler) LinkBrokerAccount(ctx context.Context, input LinkBrokerAccountInput) error {
	store.catchUp(ctx)
	account, ok := store.accountByID[input.AccountID]
	if !ok {
		return ErrAccountNotFound
	}
	userID := contextx.GetUserID(ctx)
	if account.UserID != userID {
		return ErrAccountForbidden
	}
	if account.BrokerLinked {
		return ErrBrokerAccountAlreadyLinked
	}
	if err := store.checkBrokerIsAlreadyLinked(input.BrokerAccount); err != nil {
		return err
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeBrokerAccountLinked),
		BrokerAccountLinkedEvent: &BrokerAccountLinkedEvent{
			AccountID:     input.AccountID,
			BrokerAccount: input.BrokerAccount,
		},
	})
	_, err := store.log.Append(payload)
	fatal.OnError(err)
	return nil
}

func (store *EventSourcedCommandHandler) checkBrokerIsAlreadyLinked(brokerAccount *broker.Account) error {
	switch brokerAccount.Type {
	case broker.AccountTypeTastyTrade:
		if _, isBrokerAccountAlreadyLinked := store.tastyTradeIDs[brokerAccount.ID]; isBrokerAccountAlreadyLinked {
			return ErrBrokerAccountAlreadyLinked
		}
	default:
		logger.Fatalf("unknown broker type %s", brokerAccount.Type)
	}
	return nil
}

func (store *EventSourcedCommandHandler) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.CatchUpInput{
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
	case EventTypeAccountCreated:
		return store.applyAccountCreatedEvent(ctx, frame.AccountCreatedEvent)
	case EventTypeBrokerAccountLinked:
		return store.applyBrokerAccountLinkedEvent(ctx, frame.BrokerAccountLinkedEvent)
	}
	return
}

func (store *EventSourcedCommandHandler) applyAccountCreatedEvent(ctx context.Context, event *AccountCreatedEvent) (err error) {
	store.accountByID[event.AccountID] = &Account{
		ID:     event.AccountID,
		UserID: event.UserID,
		Name:   event.AccountName,
	}
	return
}

func (store *EventSourcedCommandHandler) applyBrokerAccountLinkedEvent(ctx context.Context, event *BrokerAccountLinkedEvent) (err error) {
	account := store.accountByID[event.AccountID]
	account.BrokerLinked = true
	account.BrokerAccount = event.BrokerAccount
	switch event.BrokerAccount.Type {
	case broker.AccountTypeTastyTrade:
		store.tastyTradeIDs[event.BrokerAccount.ID] = struct{}{}
	default:
		logger.Fatalf("unknown broker type %s", event.BrokerAccount.Type)
	}
	return
}
