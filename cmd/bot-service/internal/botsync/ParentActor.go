package botsync

import (
	"context"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/logger"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

type ParentActor struct {
	log                    eventsource.Log
	botEventLogFactory     eventsource.LogFactory
	botChannelFunc         func(botID string) string
	tradingParams          tradingstrategy.Parameters
	rsiPeriod              int
	macdFastPeriod         int
	macdSlowPeriod         int
	macdSignalPeriod       int
	bollingerPeriod        int
	bollingerStdDev        float64
	sessionInterval        string
	indicatorResetInterval string

	accountClientFactory    broker.AccountClientFactory
	marketDataClientFactory broker.MarketDataClientFactory
	tradeBotByID            map[string]*TradeBot
	cancelByBotID           map[string]context.CancelFunc
}

type NewParentActorInput struct {
	Log                           eventsource.Log
	BotEventLogFactory            eventsource.LogFactory
	BotChannelFunc                func(botID string) string
	BrokerAccountClientFactory    broker.AccountClientFactory
	BrokerMarketDataClientFactory broker.MarketDataClientFactory
	TradingParams                 tradingstrategy.Parameters
	RSIPeriod                     int
	MACDFastPeriod                int
	MACDSlowPeriod                int
	MACDSignalPeriod              int
	BollingerPeriod               int
	BollingerStdDev               float64
	SessionInterval               string
	IndicatorResetInterval        string
}

func NewParentActor(input NewParentActorInput) *ParentActor {
	return &ParentActor{
		log:                     input.Log,
		botEventLogFactory:      input.BotEventLogFactory,
		botChannelFunc:          input.BotChannelFunc,
		tradingParams:           input.TradingParams,
		rsiPeriod:               input.RSIPeriod,
		macdFastPeriod:          input.MACDFastPeriod,
		macdSlowPeriod:          input.MACDSlowPeriod,
		macdSignalPeriod:        input.MACDSignalPeriod,
		bollingerPeriod:         input.BollingerPeriod,
		bollingerStdDev:         input.BollingerStdDev,
		sessionInterval:         input.SessionInterval,
		indicatorResetInterval:  input.IndicatorResetInterval,
		accountClientFactory:    input.BrokerAccountClientFactory,
		marketDataClientFactory: input.BrokerMarketDataClientFactory,
		tradeBotByID:            make(map[string]*TradeBot),
		cancelByBotID:           make(map[string]context.CancelFunc),
	}
}

type TradeBot struct {
	ID                string
	AccountID         string
	BrokerID          string
	BrokerType        string
	Symbol            string
	AllocationPercent float64
	Parameters        *tradingstrategy.Parameters
	IsActive          bool
}

func (actor *ParentActor) CatchUp(ctx context.Context) int64 {
	cursor, err := subscription.CatchUp(ctx, subscription.Input{
		Log:    actor.log,
		Cursor: 0,
		Apply:  actor.applyCatchup,
	})
	fatal.OnError(err)
	actor.startTradeActors(ctx)
	return cursor
}

func (actor *ParentActor) startTradeActors(ctx context.Context) {
	for botID, bot := range actor.tradeBotByID {
		if !bot.IsActive {
			continue
		}
		actor.startTradeActor(ctx, botID)
	}
}

func (actor *ParentActor) applyCatchup(ctx context.Context, event *eventsource.Event) (err error) {
	var frame botstore.EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case botstore.EventTypeBotCreated:
		actor.tradeBotByID[frame.BotCreatedEvent.BotID] = &TradeBot{
			ID:                frame.BotCreatedEvent.BotID,
			AccountID:         frame.BotCreatedEvent.AccountID,
			BrokerID:          frame.BotCreatedEvent.BrokerAccountID,
			BrokerType:        frame.BotCreatedEvent.BrokerType,
			Symbol:            frame.BotCreatedEvent.Symbol,
			AllocationPercent: frame.BotCreatedEvent.AllocationPercent,
			Parameters:        frame.BotCreatedEvent.TradingParameters,
			IsActive:          false,
		}
		return
	case botstore.EventTypeBotStatusUpdated:
		bot, ok := actor.tradeBotByID[frame.BotStatusUpdatedEvent.BotID]
		if !ok {
			return
		}
		bot.IsActive = frame.BotStatusUpdatedEvent.Status == botstore.BotStatusRunning
		return
	case botstore.EventTypeBotStatusDeleted:
		delete(actor.tradeBotByID, frame.BotStatusDeletedEvent.BotID)
		return
	}
	return
}

func (actor *ParentActor) Apply(ctx context.Context, event *eventsource.Event) (err error) {
	var frame botstore.EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case botstore.EventTypeBotCreated:
		return actor.applyBotCreatedEvent(ctx, frame.BotCreatedEvent)
	case botstore.EventTypeBotStatusUpdated:
		return actor.applyBotStatusUpdatedEvent(ctx, frame.BotStatusUpdatedEvent)
	case botstore.EventTypeBotStatusDeleted:
		return actor.applyBotStatusDeletedEvent(ctx, frame.BotStatusDeletedEvent)
	}
	return nil
}

func (actor *ParentActor) applyBotCreatedEvent(ctx context.Context, event *botstore.BotCreatedEvent) (err error) {
	actor.tradeBotByID[event.BotID] = &TradeBot{
		ID:                event.BotID,
		AccountID:         event.AccountID,
		BrokerID:          event.BrokerAccountID,
		BrokerType:        event.BrokerType,
		Symbol:            event.Symbol,
		AllocationPercent: event.AllocationPercent,
		Parameters:        event.TradingParameters,
		IsActive:          false,
	}
	return nil
}

func (actor *ParentActor) applyBotStatusUpdatedEvent(ctx context.Context, event *botstore.BotStatusUpdatedEvent) (err error) {
	bot, ok := actor.tradeBotByID[event.BotID]
	if !ok {
		return
	}
	bot.IsActive = event.Status == botstore.BotStatusRunning
	if bot.IsActive {
		return actor.startTradeActor(ctx, event.BotID)
	}
	return actor.stopTradeActor(ctx, event.BotID)
}

func (actor *ParentActor) applyBotStatusDeletedEvent(ctx context.Context, event *botstore.BotStatusDeletedEvent) (err error) {
	err = actor.stopTradeActor(ctx, event.BotID)
	fatal.OnError(err)
	delete(actor.tradeBotByID, event.BotID)
	return
}

func (actor *ParentActor) startTradeActor(ctx context.Context, botID string) (err error) {
	if _, isRunning := actor.cancelByBotID[botID]; isRunning {
		logger.Noticef("Trading actor for bot %s is already running; no-op start", botID)
		return nil
	}
	bot, ok := actor.tradeBotByID[botID]
	if !ok {
		return
	}
	strategy := tradingstrategy.FromParameters(bot.Parameters)
	logger.Noticef(
		"bot %s strategy config: entryMode=%s maxPosition=%.4f takeProfit=%.4f stopLoss=%.4f sessionStart=%d sessionEnd=%d minRSI=%.2f requireMACDAboveSignal=%t requireBollingerBreakout=%t minBollingerWidthPct=%.4f requireBollingerSqueeze=%t maxBollingerWidthPct=%.4f reentryCooldownMin=%d useVolatilityTP=%t volatilityTPMult=%.4f riskPerTradePct=%.4f rsiPeriod=%d macdFast=%d macdSlow=%d macdSignal=%d bollPeriod=%d bollStdDev=%.2f sessionInterval=%s indicatorResetInterval=%s",
		botID,
		bot.Parameters.EntryMode,
		bot.Parameters.MaxPositionFraction,
		bot.Parameters.TakeProfitPct,
		bot.Parameters.StopLossPct,
		bot.Parameters.SessionStart,
		bot.Parameters.SessionEnd,
		bot.Parameters.MinRSI,
		bot.Parameters.RequireMACDSignal,
		bot.Parameters.RequireBollingerBreakout,
		bot.Parameters.MinBollingerWidthPct,
		bot.Parameters.RequireBollingerSqueeze,
		bot.Parameters.MaxBollingerWidthPct,
		bot.Parameters.ReentryCooldownMinutes,
		bot.Parameters.UseVolatilityTP,
		bot.Parameters.VolatilityTPMultiplier,
		bot.Parameters.RiskPerTradePct,
		actor.rsiPeriod,
		actor.macdFastPeriod,
		actor.macdSlowPeriod,
		actor.macdSignalPeriod,
		actor.bollingerPeriod,
		actor.bollingerStdDev,
		actor.sessionInterval,
		actor.indicatorResetInterval,
	)
	ctx, cancel := context.WithCancel(ctx)
	actor.cancelByBotID[botID] = cancel
	brokerAccount := &broker.Account{
		Type: broker.AccountType(bot.BrokerType),
		ID:   bot.BrokerID,
	}
	channel := actor.botChannelFunc(botID)
	log, err := actor.botEventLogFactory.Create(channel)
	fatal.OnError(err)
	tradeActor := NewTradeActor(NewTradeActorInput{
		AccountClient:          actor.accountClientFactory.Get(ctx, brokerAccount),
		MarketDataClient:       actor.marketDataClientFactory.Get(ctx, brokerAccount),
		MarketState:            NewMarketState(bot.Symbol, actor.sessionInterval),
		TradingStrategy:        strategy,
		RSIPeriod:              actor.rsiPeriod,
		MACDFastPeriod:         actor.macdFastPeriod,
		MACDSlowPeriod:         actor.macdSlowPeriod,
		MACDSignalPeriod:       actor.macdSignalPeriod,
		BollingerPeriod:        actor.bollingerPeriod,
		BollingerStdDev:        actor.bollingerStdDev,
		IndicatorResetInterval: actor.indicatorResetInterval,
		BotID:                  botID,
		Log:                    log,
		BreakoutLookbackBars:   bot.Parameters.BreakoutLookbackBars,
	})
	logger.Noticef("Starting trading actor for bot %s", botID)
	go tradeActor.Run(ctx)
	return
}

func (actor *ParentActor) stopTradeActor(ctx context.Context, botID string) (err error) {
	cancel, ok := actor.cancelByBotID[botID]
	if !ok {
		return
	}
	logger.Noticef("Stopping trading actor for bot %s", botID)
	cancel()
	delete(actor.cancelByBotID, botID)
	return nil
}
