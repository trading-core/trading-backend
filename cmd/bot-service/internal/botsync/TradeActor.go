package botsync

import (
	"context"
	"sync"
	"time"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/logger"
)

const accountSnapshotRefreshInterval = 1 * time.Second

type TradeActor struct {
	botID            string
	accountClient    broker.AccountClient
	tradingStrategy  tradingstrategy.Strategy
	marketDataClient broker.MarketDataClient
	marketState      *MarketState
	log              eventsource.Log

	mutex              sync.RWMutex
	accountSnapshot    tradingstrategy.AccountSnapshot
	hasAccountSnapshot bool
	orderPlaced        bool // optimistic HasOpenOrder guard until broker reflects the order
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
		decision := actor.tradingStrategy.Evaluate(input)
		if decision.Action == tradingstrategy.ActionNone {
			continue
		}
		var orderAction broker.OrderAction
		if decision.Action == tradingstrategy.ActionBuy {
			orderAction = broker.OrderActionBuy
		} else {
			orderAction = broker.OrderActionSell
		}
		_, err := actor.accountClient.PlaceOrder(ctx, broker.PlaceOrderInput{
			Symbol:   actor.marketState.Symbol(),
			Action:   orderAction,
			Quantity: decision.Quantity,
		})
		if err != nil {
			logger.Warnf("bot %s: failed to place %s order: %v", actor.botID, orderAction, err)
		} else {
			actor.mutex.Lock()
			actor.accountSnapshot.HasOpenOrder = true
			actor.orderPlaced = true
			actor.mutex.Unlock()
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
	symbol := actor.marketState.Symbol()
	position, err := accountClient.GetEquityPosition(ctx, symbol)
	if err != nil {
		return
	}
	hasPendingOrder, err := accountClient.HasPendingOrder(ctx, symbol)
	if err != nil {
		return
	}
	snapshot = tradingstrategy.AccountSnapshot{
		CashBalance:      balance.CashBalance,
		BuyingPower:      balance.EquityBuyingPower,
		PositionQuantity: position.Quantity,
		HasOpenOrder:     hasPendingOrder,
	}
	return
}

func (actor *TradeActor) startAccountSnapshotRefresher(ctx context.Context) {
	refresh := func() {
		snapshot, err := actor.loadAccountSnapshot(ctx, actor.accountClient)
		actor.mutex.Lock()
		defer actor.mutex.Unlock()
		if err != nil {
			logger.Warnf("bot %s: failed to refresh account snapshot: %v", actor.botID, err)
			return
		}
		// If the broker now shows a pending order, clear the optimistic flag.
		// If not yet reflected but we placed one, keep HasOpenOrder true.
		if actor.orderPlaced && !snapshot.HasOpenOrder {
			snapshot.HasOpenOrder = true
		} else {
			actor.orderPlaced = false
		}
		actor.accountSnapshot = snapshot
		actor.hasAccountSnapshot = true
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
