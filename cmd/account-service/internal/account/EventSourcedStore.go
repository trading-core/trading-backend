package account

import (
	"context"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ Store = (*EventSourcedStore)(nil)

type EventSourcedStore struct {
	log                    eventsource.Log
	cursor                 int64
	accountByID            map[string]*Account
	brokerAccountIDsByType map[string]map[string]struct{}
}

type NewEventSourcedStoreInput struct {
	Log eventsource.Log
}

func NewEventSourcedStore(input NewEventSourcedStoreInput) *EventSourcedStore {
	return &EventSourcedStore{
		log:                    input.Log,
		accountByID:            make(map[string]*Account),
		brokerAccountIDsByType: make(map[string]map[string]struct{}),
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
	if _, ok := store.brokerAccountIDsByType[input.BrokerAccount.Type][input.BrokerAccount.ID]; ok {
		return ErrBrokerAccountAlreadyLinked
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
	var accounts []*Account
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
		store.accountByID[frame.AccountCreatedEvent.AccountID] = &Account{
			ID:     frame.AccountCreatedEvent.AccountID,
			UserID: frame.AccountCreatedEvent.UserID,
			Name:   frame.AccountCreatedEvent.AccountName,
		}
	case EventTypeBrokerAccountLinked:
		account := store.accountByID[frame.BrokerAccountLinkedEvent.AccountID]
		account.BrokerLinked = true
		account.BrokerAccount = frame.BrokerAccountLinkedEvent.BrokerAccount
		if _, ok := store.brokerAccountIDsByType[frame.BrokerAccountLinkedEvent.BrokerAccount.Type]; !ok {
			store.brokerAccountIDsByType[frame.BrokerAccountLinkedEvent.BrokerAccount.Type] = make(map[string]struct{})
		}
		store.brokerAccountIDsByType[frame.BrokerAccountLinkedEvent.BrokerAccount.Type][frame.BrokerAccountLinkedEvent.BrokerAccount.ID] = struct{}{}
	}
	return
}
