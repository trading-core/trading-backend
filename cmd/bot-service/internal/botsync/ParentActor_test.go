package botsync

import (
	"context"
	"sync"
	"testing"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/iterator"
	. "github.com/smartystreets/goconvey/convey"
)

func TestParentActorStatusUpdates(t *testing.T) {
	Convey("Given a parent actor with one created bot", t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		accountFactory := &fakeAccountClientFactory{client: fakeAccountClient{}}
		marketFactory := &fakeMarketDataClientFactory{client: fakeMarketDataClient{}}

		actor := NewParentActor(NewParentActorInput{
			Log:                           eventsource.NewInMemoryLog("bot-control"),
			BotEventLogFactory:            eventsource.NewInMemoryLogFactory(),
			BotChannelFunc:                func(botID string) string { return "bot:" + botID + ":events" },
			BrokerAccountClientFactory:    accountFactory,
			BrokerMarketDataClientFactory: marketFactory,
		})

		err := actor.applyBotCreatedEvent(ctx, &botstore.BotCreatedEvent{
			BotID:             "bot-1",
			AccountID:         "acct-1",
			BrokerAccountID:   "broker-1",
			BrokerType:        string(broker.AccountTypeTastyTrade),
			Symbol:            "AAPL",
			StrategyTradeType: "scalping",
		})
		So(err, ShouldBeNil)

		Convey("When RUNNING is applied twice", func() {
			err = actor.applyBotStatusUpdatedEvent(ctx, &botstore.BotStatusUpdatedEvent{
				BotID:  "bot-1",
				Status: botstore.BotStatusRunning,
			})
			So(err, ShouldBeNil)

			err = actor.applyBotStatusUpdatedEvent(ctx, &botstore.BotStatusUpdatedEvent{
				BotID:  "bot-1",
				Status: botstore.BotStatusRunning,
			})
			So(err, ShouldBeNil)

			Convey("Then start is a no-op on the second update", func() {
				So(accountFactory.Calls(), ShouldEqual, 1)
				So(marketFactory.Calls(), ShouldEqual, 1)
				So(len(actor.cancelByBotID), ShouldEqual, 1)
			})
		})

		Convey("When bot transitions RUNNING then STOPPED then RUNNING", func() {
			err = actor.applyBotStatusUpdatedEvent(ctx, &botstore.BotStatusUpdatedEvent{
				BotID:  "bot-1",
				Status: botstore.BotStatusRunning,
			})
			So(err, ShouldBeNil)

			err = actor.applyBotStatusUpdatedEvent(ctx, &botstore.BotStatusUpdatedEvent{
				BotID:  "bot-1",
				Status: botstore.BotStatusStopped,
			})
			So(err, ShouldBeNil)

			err = actor.applyBotStatusUpdatedEvent(ctx, &botstore.BotStatusUpdatedEvent{
				BotID:  "bot-1",
				Status: botstore.BotStatusRunning,
			})
			So(err, ShouldBeNil)

			Convey("Then actor restarts and factory Get is called twice overall", func() {
				So(accountFactory.Calls(), ShouldEqual, 2)
				So(marketFactory.Calls(), ShouldEqual, 2)
				So(len(actor.cancelByBotID), ShouldEqual, 1)
			})
		})

		_ = actor.stopTradeActor(ctx, "bot-1")
	})
}

type fakeAccountClientFactory struct {
	mu     sync.Mutex
	calls  int
	client broker.AccountClient
}

func (f *fakeAccountClientFactory) Get(ctx context.Context, account *broker.Account) broker.AccountClient {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.client
}

func (f *fakeAccountClientFactory) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type fakeMarketDataClientFactory struct {
	mu     sync.Mutex
	calls  int
	client broker.MarketDataClient
}

func (f *fakeMarketDataClientFactory) Get(ctx context.Context, account *broker.Account) broker.MarketDataClient {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.client
}

func (f *fakeMarketDataClientFactory) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type fakeAccountClient struct{}

func (fakeAccountClient) GetBalance(ctx context.Context) (*broker.GetBalanceOutput, error) {
	return &broker.GetBalanceOutput{
		CashBalance:       1000,
		EquityBuyingPower: 1000,
	}, nil
}

func (fakeAccountClient) GetEquityPosition(ctx context.Context, symbol string) (*broker.GetEquityPositionOutput, error) {
	return &broker.GetEquityPositionOutput{Quantity: 0}, nil
}

func (fakeAccountClient) PlaceOrder(ctx context.Context, input broker.PlaceOrderInput) (*broker.PlaceOrderOutput, error) {
	return &broker.PlaceOrderOutput{OrderID: 1}, nil
}

func (fakeAccountClient) HasPendingOrder(ctx context.Context, symbol string) (bool, error) {
	return false, nil
}

type fakeMarketDataClient struct{}

func (fakeMarketDataClient) Stream(ctx context.Context, input broker.StreamMarketDataInput) iterator.Iterator[*broker.MarketDataMessage] {
	return emptyMarketDataIterator{}
}

type emptyMarketDataIterator struct{}

func (emptyMarketDataIterator) Next() bool {
	return false
}

func (emptyMarketDataIterator) Item() *broker.MarketDataMessage {
	return nil
}

func (emptyMarketDataIterator) Err() error {
	return nil
}
