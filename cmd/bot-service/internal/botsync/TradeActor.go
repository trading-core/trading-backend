package botsync

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/logger"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

const accountSnapshotRefreshInterval = 1 * time.Second

// orderPlacedTTL is the number of account-snapshot refresh cycles the
// optimistic HasOpenOrder guard stays active. If the broker hasn't reported
// the order as pending within this window (e.g. it filled instantly), the
// guard expires so the bot isn't stuck forever.
const orderPlacedTTL = 3

// orderFailureCooldown prevents rapid-fire PlaceOrder retries when the
// broker returns transient errors.
const orderFailureCooldown = 5 * time.Second

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
	entryPrice         float64   // persisted via decision events in the bot's event log
	orderGuardTTL      int       // countdown refreshes for optimistic HasOpenOrder
	orderFailedUntil   time.Time // suppress new orders until this time
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
	actor.restoreStrategyState(ctx)
	actor.startAccountSnapshotRefresher(ctx)
	iterator := actor.marketDataClient.Stream(ctx, broker.StreamMarketDataInput{
		Symbol: actor.marketState.Symbol(),
	})
	for iterator.Next() {
		if ctx.Err() != nil {
			break
		}
		accountSnapshot, ok := actor.getAccountSnapshot()
		if !ok {
			continue
		}
		message := iterator.Item()
		snapshot := actor.marketState.Apply(message)
		accountSnapshot.EntryPrice = actor.entryPrice
		input := tradingstrategy.NewEvaluateInput(snapshot, accountSnapshot)
		decision := actor.tradingStrategy.Evaluate(input)
		if decision.Action == tradingstrategy.ActionNone {
			continue
		}
		if actor.isInOrderCooldown() {
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
			actor.mutex.Lock()
			actor.orderFailedUntil = time.Now().Add(orderFailureCooldown)
			actor.mutex.Unlock()
			continue
		}
		actor.mutex.Lock()
		actor.accountSnapshot.HasOpenOrder = true
		actor.orderGuardTTL = orderPlacedTTL
		if decision.Action == tradingstrategy.ActionBuy {
			actor.entryPrice = input.Price
		} else if decision.Action == tradingstrategy.ActionSell {
			actor.entryPrice = 0
		}
		actor.mutex.Unlock()
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
		// Optimistic guard: if we recently placed an order and the broker
		// hasn't reflected it yet, keep HasOpenOrder true. The TTL counter
		// prevents this from getting stuck if the order filled instantly.
		if actor.orderGuardTTL > 0 {
			if snapshot.HasOpenOrder {
				// Broker caught up — stop overriding.
				actor.orderGuardTTL = 0
			} else {
				snapshot.HasOpenOrder = true
				actor.orderGuardTTL--
			}
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

func (actor *TradeActor) isInOrderCooldown() bool {
	actor.mutex.RLock()
	defer actor.mutex.RUnlock()
	return time.Now().Before(actor.orderFailedUntil)
}

// restoreStrategyState replays past decision events from the bot's event log
// to recover entry price after a restart.
func (actor *TradeActor) restoreStrategyState(ctx context.Context) {
	_, err := subscription.CatchUp(ctx, subscription.Input{
		Log:    actor.log,
		Cursor: 0,
		Apply: func(ctx context.Context, event *eventsource.Event) error {
			var frame EventFrame
			if err := json.Unmarshal(event.Data, &frame); err != nil {
				return nil // skip malformed events
			}
			if frame.Type != EventTypeBotDecisionRecorded || frame.BotDecisionRecordedEvent == nil {
				return nil
			}
			d := frame.BotDecisionRecordedEvent
			switch tradingstrategy.Action(d.Action) {
			case tradingstrategy.ActionBuy:
				actor.entryPrice = d.Price
			case tradingstrategy.ActionSell:
				actor.entryPrice = 0
			}
			return nil
		},
	})
	if err != nil {
		logger.Warnf("bot %s: failed to restore strategy state: %v", actor.botID, err)
	}
}
