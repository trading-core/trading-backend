package botsync

import (
	"context"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/logger"
)

type ParentActor struct {
	log                     eventsource.Log
	accountClientFactory    broker.AccountClientFactory
	marketDataClientFactory broker.MarketDataClientFactory
	tradeBotByID            map[string]*TradeBot
	cancelByBotID           map[string]context.CancelFunc
}

type NewParentActorInput struct {
	Log                           eventsource.Log
	BrokerAccountClientFactory    broker.AccountClientFactory
	BrokerMarketDataClientFactory broker.MarketDataClientFactory
}

func NewParentActor(input NewParentActorInput) *ParentActor {
	return &ParentActor{
		log:                     input.Log,
		accountClientFactory:    input.BrokerAccountClientFactory,
		marketDataClientFactory: input.BrokerMarketDataClientFactory,
		tradeBotByID:            make(map[string]*TradeBot),
		cancelByBotID:           make(map[string]context.CancelFunc),
	}
}

type TradeBot struct {
	ID                string
	BrokerID          string
	BrokerType        string
	Symbol            string
	StrategyType      string
	AllocationPercent float64
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
			BrokerID:          frame.BotCreatedEvent.BrokerAccountID,
			BrokerType:        frame.BotCreatedEvent.BrokerType,
			Symbol:            frame.BotCreatedEvent.Symbol,
			StrategyType:      frame.BotCreatedEvent.StrategyTradeType,
			AllocationPercent: frame.BotCreatedEvent.AllocationPercent,
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
		BrokerID:          event.BrokerAccountID,
		BrokerType:        event.BrokerType,
		Symbol:            event.Symbol,
		StrategyType:      event.StrategyTradeType,
		AllocationPercent: event.AllocationPercent,
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
	bot, ok := actor.tradeBotByID[botID]
	if !ok {
		return
	}
	strategy := tradingstrategy.New(bot.StrategyType)
	err = tradingstrategy.Validate(strategy)
	fatal.OnError(err)
	ctx, cancel := context.WithCancel(ctx)
	actor.cancelByBotID[botID] = cancel
	brokerAccount := &broker.Account{
		Type: broker.AccountType(bot.BrokerType),
		ID:   bot.BrokerID,
	}
	tradeActor := NewTradeActor(NewTradeActorInput{
		AccountClient:    actor.accountClientFactory.Get(ctx, brokerAccount),
		MarketDataClient: actor.marketDataClientFactory.Get(ctx, brokerAccount),
		MarketState:      NewMarketState(bot.Symbol),
		TradingStrategy:  strategy,
		BotID:            botID,
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
	cancel()
	delete(actor.cancelByBotID, botID)
	return nil
}
