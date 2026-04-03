package botsync

import (
	"context"
	"encoding/json"
	"math"
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
	indicators       *indicatorState

	mutex              sync.RWMutex
	accountSnapshot    tradingstrategy.AccountSnapshot
	hasAccountSnapshot bool
	entryPrice         float64    // persisted via decision events in the bot's event log
	highSinceEntry     float64    // highest price observed since entry (trailing stop)
	lastStopLossAt     *time.Time // time of last trailing-stop exit (re-entry cooldown)
	orderGuardTTL      int        // countdown refreshes for optimistic HasOpenOrder
	orderFailedUntil   time.Time  // suppress new orders until this time
}

type NewTradeActorInput struct {
	AccountClient    broker.AccountClient
	MarketDataClient broker.MarketDataClient
	MarketState      *MarketState
	TradingStrategy  tradingstrategy.Strategy
	RSIPeriod        int
	MACDFastPeriod   int
	MACDSlowPeriod   int
	MACDSignalPeriod int
	BollingerPeriod  int
	BollingerStdDev  float64
	BotID            string
	Log              eventsource.Log
}

func NewTradeActor(input NewTradeActorInput) *TradeActor {
	return &TradeActor{
		accountClient:    input.AccountClient,
		marketDataClient: input.MarketDataClient,
		tradingStrategy:  input.TradingStrategy,
		marketState:      input.MarketState,
		indicators:       newIndicatorState(input.RSIPeriod, input.MACDFastPeriod, input.MACDSlowPeriod, input.MACDSignalPeriod, input.BollingerPeriod, input.BollingerStdDev),
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
		rsi, macd, macdSignal, bollUpper, bollMiddle, bollLower, bollWidthPct := actor.indicators.Update(input.Price)
		input.RSI = rsi
		input.MACD = macd
		input.MACDSignal = macdSignal
		input.BollUpper = bollUpper
		input.BollMiddle = bollMiddle
		input.BollLower = bollLower
		input.BollWidthPct = bollWidthPct
		// Track trailing high while in position.
		if accountSnapshot.PositionQuantity > 0 && input.Price > actor.highSinceEntry {
			actor.highSinceEntry = input.Price
		}
		input.HighSinceEntry = actor.highSinceEntry
		input.LastStopLossAt = actor.lastStopLossAt
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
			actor.highSinceEntry = input.Price
		} else if decision.Action == tradingstrategy.ActionSell {
			actor.entryPrice = 0
			actor.highSinceEntry = 0
			if decision.Reason == "trailing stop triggered" {
				now := time.Now()
				actor.lastStopLossAt = &now
			}
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

type indicatorState struct {
	rsiPeriod      int
	macdFastPeriod int
	macdSlowPeriod int
	macdSignal     int
	bollPeriod     int
	bollStdDev     float64

	priceSamples int
	prevClose    *float64

	gainSum      float64
	lossSum      float64
	rsiSeedCount int
	rsiReady     bool
	avgGain      float64
	avgLoss      float64

	fastEMA      float64
	slowEMA      float64
	macdSeed     []float64
	signalEMA    float64
	signalReady  bool
	hasEMAValues bool

	bollWindow []float64
	bollSum    float64
	bollSqSum  float64
}

func newIndicatorState(rsiPeriod int, macdFastPeriod int, macdSlowPeriod int, macdSignal int, bollPeriod int, bollStdDev float64) *indicatorState {
	if rsiPeriod < 2 {
		rsiPeriod = 14
	}
	if macdFastPeriod < 2 {
		macdFastPeriod = 12
	}
	if macdSlowPeriod <= macdFastPeriod {
		macdSlowPeriod = 26
	}
	if macdSignal < 2 {
		macdSignal = 9
	}
	if bollPeriod < 2 {
		bollPeriod = 20
	}
	if bollStdDev <= 0 {
		bollStdDev = 2.0
	}
	return &indicatorState{
		rsiPeriod:      rsiPeriod,
		macdFastPeriod: macdFastPeriod,
		macdSlowPeriod: macdSlowPeriod,
		macdSignal:     macdSignal,
		bollPeriod:     bollPeriod,
		bollStdDev:     bollStdDev,
	}
}

func (state *indicatorState) Update(price float64) (rsi *float64, macd *float64, macdSignal *float64, bollUpper *float64, bollMiddle *float64, bollLower *float64, bollWidthPct *float64) {
	if price <= 0 {
		return nil, nil, nil, nil, nil, nil, nil
	}
	state.priceSamples++

	if !state.hasEMAValues {
		state.fastEMA = price
		state.slowEMA = price
		state.hasEMAValues = true
	} else {
		fastK := 2.0 / (float64(state.macdFastPeriod) + 1)
		slowK := 2.0 / (float64(state.macdSlowPeriod) + 1)
		state.fastEMA = ((price - state.fastEMA) * fastK) + state.fastEMA
		state.slowEMA = ((price - state.slowEMA) * slowK) + state.slowEMA
	}
	macdValue := state.fastEMA - state.slowEMA
	if state.priceSamples >= state.macdSlowPeriod {
		v := macdValue
		macd = &v
		if !state.signalReady {
			state.macdSeed = append(state.macdSeed, macdValue)
			if len(state.macdSeed) >= state.macdSignal {
				sum := 0.0
				for _, sample := range state.macdSeed {
					sum += sample
				}
				state.signalEMA = sum / float64(len(state.macdSeed))
				state.signalReady = true
			}
		} else {
			signalK := 2.0 / (float64(state.macdSignal) + 1)
			state.signalEMA = ((macdValue - state.signalEMA) * signalK) + state.signalEMA
		}
		if state.signalReady {
			v := state.signalEMA
			macdSignal = &v
		}
	}

	if state.prevClose != nil {
		delta := price - *state.prevClose
		gain := 0.0
		loss := 0.0
		if delta > 0 {
			gain = delta
		} else {
			loss = -delta
		}
		if !state.rsiReady {
			state.gainSum += gain
			state.lossSum += loss
			state.rsiSeedCount++
			if state.rsiSeedCount >= state.rsiPeriod {
				state.avgGain = state.gainSum / float64(state.rsiPeriod)
				state.avgLoss = state.lossSum / float64(state.rsiPeriod)
				state.rsiReady = true
				rsiValue := rsiFromAverages(state.avgGain, state.avgLoss)
				rsi = &rsiValue
			}
		} else {
			state.avgGain = ((state.avgGain * float64(state.rsiPeriod-1)) + gain) / float64(state.rsiPeriod)
			state.avgLoss = ((state.avgLoss * float64(state.rsiPeriod-1)) + loss) / float64(state.rsiPeriod)
			rsiValue := rsiFromAverages(state.avgGain, state.avgLoss)
			rsi = &rsiValue
		}
	}

	state.bollWindow = append(state.bollWindow, price)
	state.bollSum += price
	state.bollSqSum += price * price
	if len(state.bollWindow) > state.bollPeriod {
		old := state.bollWindow[0]
		state.bollWindow = state.bollWindow[1:]
		state.bollSum -= old
		state.bollSqSum -= old * old
	}
	if len(state.bollWindow) == state.bollPeriod {
		mean := state.bollSum / float64(state.bollPeriod)
		variance := (state.bollSqSum / float64(state.bollPeriod)) - (mean * mean)
		if variance < 0 {
			variance = 0
		}
		stddev := math.Sqrt(variance)
		upper := mean + (state.bollStdDev * stddev)
		lower := mean - (state.bollStdDev * stddev)
		mid := mean
		width := 0.0
		if mid != 0 {
			width = (upper - lower) / mid
		}
		bollUpper = &upper
		bollMiddle = &mid
		bollLower = &lower
		bollWidthPct = &width
	}
	closeCopy := price
	state.prevClose = &closeCopy
	return rsi, macd, macdSignal, bollUpper, bollMiddle, bollLower, bollWidthPct
}

func rsiFromAverages(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
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
