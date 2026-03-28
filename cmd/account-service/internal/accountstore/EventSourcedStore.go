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

var _ Store = (*EventSourcedStore)(nil)

type EventSourcedStore struct {
	log           eventsource.Log
	cursor        int64
	accountByID   map[string]*Account
	tastyTradeIDs map[string]struct{}
}

type NewEventSourcedStoreInput struct {
	Log eventsource.Log
}

func NewEventSourcedStore(input NewEventSourcedStoreInput) *EventSourcedStore {
	return &EventSourcedStore{
		log:           input.Log,
		accountByID:   make(map[string]*Account),
		tastyTradeIDs: make(map[string]struct{}),
	}
}

func (store *EventSourcedStore) Create(ctx context.Context, input CreateInput) error {
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

func (store *EventSourcedStore) LinkBrokerAccount(ctx context.Context, input LinkBrokerAccountInput) error {
	store.catchUp(ctx)
	account, ok := store.accountByID[input.AccountID]
	if !ok {
		return ErrNotFound
	}
	userID := contextx.GetUserID(ctx)
	if account.UserID != userID {
		return ErrForbidden
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

func (store *EventSourcedStore) checkBrokerIsAlreadyLinked(brokerAccount *broker.Account) error {
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

func (store *EventSourcedStore) Get(ctx context.Context, input GetInput) (*Account, error) {
	store.catchUp(ctx)
	account, ok := store.accountByID[input.AccountID]
	if !ok {
		return nil, ErrNotFound
	}
	userID := contextx.GetUserID(ctx)
	if account.UserID != userID {
		return nil, ErrForbidden
	}
	return account, nil
}

func (store *EventSourcedStore) List(ctx context.Context) ([]*Account, error) {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	accounts := make([]*Account, 0)
	for _, account := range store.accountByID {
		if account.UserID != userID {
			continue
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
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
	case EventTypeAccountCreated:
		store.applyAccountCreatedEvent(ctx, frame)
	case EventTypeBrokerAccountLinked:
		store.applyBrokerAccountLinkedEvent(ctx, frame)
	}
	return
}

func (store *EventSourcedStore) applyAccountCreatedEvent(ctx context.Context, frame EventFrame) (err error) {
	store.accountByID[frame.AccountCreatedEvent.AccountID] = &Account{
		ID:     frame.AccountCreatedEvent.AccountID,
		UserID: frame.AccountCreatedEvent.UserID,
		Name:   frame.AccountCreatedEvent.AccountName,
	}
	return
}

func (store *EventSourcedStore) applyBrokerAccountLinkedEvent(ctx context.Context, frame EventFrame) (err error) {
	account := store.accountByID[frame.BrokerAccountLinkedEvent.AccountID]
	account.BrokerLinked = true
	account.BrokerAccount = frame.BrokerAccountLinkedEvent.BrokerAccount
	switch frame.BrokerAccountLinkedEvent.BrokerAccount.Type {
	case broker.AccountTypeTastyTrade:
		store.tastyTradeIDs[frame.BrokerAccountLinkedEvent.BrokerAccount.ID] = struct{}{}
	default:
		logger.Fatalf("unknown broker type %s", frame.BrokerAccountLinkedEvent.BrokerAccount.Type)
	}
	return
}
