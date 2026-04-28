package accountstore

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
	log         eventsource.Log
	cursor      int64
	accountByID map[string]*Account
}

type NewEventSourcedQueryHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedQueryHandler(input NewEventSourcedQueryHandlerInput) *EventSourcedQueryHandler {
	return &EventSourcedQueryHandler{
		log:         input.Log,
		accountByID: make(map[string]*Account),
	}
}

func (store *EventSourcedQueryHandler) Get(ctx context.Context, input GetInput) (*Account, error) {
	store.catchUp(ctx)
	account, ok := store.accountByID[input.AccountID]
	if !ok {
		return nil, ErrAccountNotFound
	}
	userID := contextx.GetUserID(ctx)
	if account.UserID != userID {
		return nil, ErrAccountForbidden
	}
	return account, nil
}

func (store *EventSourcedQueryHandler) List(ctx context.Context) ([]*Account, error) {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	accounts := make([]*Account, 0)
	for _, account := range store.accountByID {
		if account.UserID != userID {
			continue
		}
		accounts = append(accounts, account)
	}
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Name < accounts[j].Name
	})
	return accounts, nil
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
	case EventTypeAccountCreated:
		return store.applyAccountCreatedEvent(ctx, frame.AccountCreatedEvent)
	case EventTypeBrokerAccountLinked:
		return store.applyBrokerAccountLinkedEvent(ctx, frame.BrokerAccountLinkedEvent)
	}
	return
}

func (store *EventSourcedQueryHandler) applyAccountCreatedEvent(ctx context.Context, event *AccountCreatedEvent) (err error) {
	store.accountByID[event.AccountID] = &Account{
		ID:     event.AccountID,
		UserID: event.UserID,
		Name:   event.AccountName,
	}
	return
}

func (store *EventSourcedQueryHandler) applyBrokerAccountLinkedEvent(ctx context.Context, event *BrokerAccountLinkedEvent) (err error) {
	account := store.accountByID[event.AccountID]
	account.BrokerLinked = true
	account.BrokerAccount = event.BrokerAccount
	return
}
