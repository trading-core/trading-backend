package botsync

import (
	"context"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
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
	actor.startTradeBots(ctx)
	return cursor
}

func (actor *ParentActor) startTradeBots(ctx context.Context) {
	for botID, bot := range actor.tradeBotByID {
		if !bot.IsActive {
			continue
		}
		err := actor.startTradeBot(ctx, botID)
		fatal.OnError(err)
	}
}

func (actor *ParentActor) applyCatchup(ctx context.Context, event *eventsource.Event) (err error) {
	var frame botstore.EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case botstore.EventTypeBotCreated:
		actor.tradeBotByID[frame.BotCreatedEvent.BotID] = &TradeBot{
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
		return actor.startTradeBot(ctx, event.BotID)
	}
	return actor.stopTradeBot(ctx, event.BotID)
}

func (actor *ParentActor) applyBotStatusDeletedEvent(ctx context.Context, event *botstore.BotStatusDeletedEvent) (err error) {
	err = actor.stopTradeBot(ctx, event.BotID)
	fatal.OnError(err)
	delete(actor.tradeBotByID, event.BotID)
	return
}

func (actor *ParentActor) startTradeBot(ctx context.Context, botID string) (err error) {
	bot, ok := actor.tradeBotByID[botID]
	if !ok {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	actor.cancelByBotID[botID] = cancel
	if actor.accountClientFactory == nil || actor.marketDataClientFactory == nil {
		return nil
	}

	strategy := tradingstrategy.New(bot.StrategyType)
	if err = tradingstrategy.Validate(strategy); err != nil {
		cancel()
		delete(actor.cancelByBotID, botID)
		return err
	}

	tradeActor := NewActor(NewTradeActorInput{
		TradingStrategy: strategy,
		Symbol:          bot.Symbol,
	})
	brokerAccount := &broker.Account{
		Type: broker.AccountType(bot.BrokerType),
		ID:   bot.BrokerID,
	}
	accountClient := actor.accountClientFactory.Get(ctx, brokerAccount)
	marketDataClient := actor.marketDataClientFactory.Get(ctx, brokerAccount)
	go actor.runTradeBot(ctx, botID, tradeActor, bot, accountClient, marketDataClient)

	return nil
}

func (actor *ParentActor) stopTradeBot(ctx context.Context, botID string) (err error) {
	cancel, ok := actor.cancelByBotID[botID]
	if !ok {
		return
	}
	cancel()
	delete(actor.cancelByBotID, botID)
	return nil
}

func (actor *ParentActor) runTradeBot(ctx context.Context, botID string, tradeActor *Actor, bot *TradeBot, accountClient broker.AccountClient, marketDataClient broker.MarketDataClient) {
	iterator := marketDataClient.Stream(ctx, broker.StreamMarketDataInput{Symbol: bot.Symbol})
	for iterator.Next() {
		accountSnapshot, err := loadAccountSnapshot(ctx, accountClient)
		if err != nil {
			continue
		}
		decision, err := tradeActor.ApplyMarketData(ctx, iterator.Item(), accountSnapshot)
		if err != nil {
			continue
		}
		actor.handleDecision(botID, decision)
	}
	_ = iterator.Err()
}

func loadAccountSnapshot(ctx context.Context, accountClient broker.AccountClient) (tradingstrategy.AccountSnapshot, error) {
	balance, err := accountClient.GetBalance(ctx)
	if err != nil {
		return tradingstrategy.AccountSnapshot{}, err
	}
	return tradingstrategy.AccountSnapshot{
		CashBalance:      balance.CashBalance,
		BuyingPower:      balance.EquityBuyingPower,
		PositionQuantity: 0,
		HasOpenOrder:     false,
	}, nil
}

func (actor *ParentActor) handleDecision(botID string, decision tradingstrategy.Decision) {
	_ = botID
	_ = decision
}
