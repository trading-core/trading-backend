package botsync

import (
	"context"
	"sync"
	"time"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
)

const accountSnapshotRefreshInterval = 1 * time.Second

type TradeActor struct {
	botID            string
	accountClient    broker.AccountClient
	tradingStrategy  tradingstrategy.Strategy
	marketDataClient broker.MarketDataClient
	marketState      *MarketState
	log              eventsource.Log

	mutex                sync.RWMutex
	accountSnapshot      tradingstrategy.AccountSnapshot
	hasAccountSnapshot   bool
	accountSnapshotError error
}

type NewTradeActorInput struct {
	AccountClient    broker.AccountClient
	MarketDataClient broker.MarketDataClient
	MarketState      *MarketState
	TradingStrategy  tradingstrategy.Strategy
	BotID            string
	Log              eventsource.Log
}

func NewTradeActor(input NewTradeActorInput) *TradeActor {
	return &TradeActor{
		accountClient:    input.AccountClient,
		marketDataClient: input.MarketDataClient,
		tradingStrategy:  input.TradingStrategy,
		marketState:      input.MarketState,
		botID:            input.BotID,
		log:              input.Log,
	}
}

func (actor *TradeActor) Run(ctx context.Context) {
	actor.startAccountSnapshotRefresher(ctx)
	iterator := actor.marketDataClient.Stream(ctx, broker.StreamMarketDataInput{
		Symbol: actor.marketState.Symbol(),
	})
	for iterator.Next() {
		accountSnapshot, ok := actor.getAccountSnapshot()
		if !ok {
			continue
		}
		message := iterator.Item()
		snapshot := actor.marketState.Apply(message)
		input := tradingstrategy.NewEvaluateInput(snapshot, accountSnapshot)
		decision, err := actor.tradingStrategy.Evaluate(input)
		if err != nil {
			continue
		}
		if decision.Action == tradingstrategy.ActionNone {
			continue
		}
		payload := fatal.UnlessMarshal(EventFrame{
			EventBase: eventsource.NewEventBase(EventTypeBotDecisionRecorded),
			BotDecisionRecordedEvent: &BotDecisionRecordedEvent{
				BotID:        actor.botID,
				Symbol:       actor.marketState.Symbol(),
				StrategyType: string(actor.tradingStrategy.Type()),
				Action:       string(decision.Action),
				Reason:       decision.Reason,
				Quantity:     decision.Quantity,
				Price:        input.Price,
			},
		})
		_, err = actor.log.Append(payload)
		fatal.OnError(err)
	}
	fatal.OnError(iterator.Err())
}

func (actor *TradeActor) loadAccountSnapshot(ctx context.Context, accountClient broker.AccountClient) (snapshot tradingstrategy.AccountSnapshot, err error) {
	balance, err := accountClient.GetBalance(ctx)
	if err != nil {
		return
	}
	snapshot = tradingstrategy.AccountSnapshot{
		CashBalance:      balance.CashBalance,
		BuyingPower:      balance.EquityBuyingPower,
		PositionQuantity: 0,
		HasOpenOrder:     false,
	}
	return
}

func (actor *TradeActor) startAccountSnapshotRefresher(ctx context.Context) {
	refresh := func() {
		snapshot, err := actor.loadAccountSnapshot(ctx, actor.accountClient)
		actor.mutex.Lock()
		defer actor.mutex.Unlock()
		if err != nil {
			actor.accountSnapshotError = err
			return
		}
		actor.accountSnapshot = snapshot
		actor.hasAccountSnapshot = true
		actor.accountSnapshotError = nil
	}
	refresh()
	go func() {
		ticker := time.NewTicker(accountSnapshotRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh()
			}
		}
	}()
}

func (actor *TradeActor) getAccountSnapshot() (tradingstrategy.AccountSnapshot, bool) {
	actor.mutex.RLock()
	defer actor.mutex.RUnlock()
	if !actor.hasAccountSnapshot {
		return tradingstrategy.AccountSnapshot{}, false
	}
	return actor.accountSnapshot, true
}
