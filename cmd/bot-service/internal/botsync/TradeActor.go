package botsync

import (
	"context"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/fatal"
)

type TradeActor struct {
	botID            string
	accountClient    broker.AccountClient
	tradingStrategy  tradingstrategy.Strategy
	marketDataClient broker.MarketDataClient
	marketState      *MarketState
}

type NewTradeActorInput struct {
	AccountClient    broker.AccountClient
	MarketDataClient broker.MarketDataClient
	MarketState      *MarketState
	TradingStrategy  tradingstrategy.Strategy
	BotID            string
}

func NewTradeActor(input NewTradeActorInput) *TradeActor {
	return &TradeActor{
		accountClient:    input.AccountClient,
		marketDataClient: input.MarketDataClient,
		tradingStrategy:  input.TradingStrategy,
		marketState:      input.MarketState,
		botID:            input.BotID,
	}
}

func (actor *TradeActor) Run(ctx context.Context) {
	iterator := actor.marketDataClient.Stream(ctx, broker.StreamMarketDataInput{
		Symbol: actor.marketState.Symbol(),
	})
	for iterator.Next() {
		accountSnapshot, err := actor.loadAccountSnapshot(ctx, actor.accountClient)
		if err != nil {
			continue
		}
		item := iterator.Item()
		decision, err := actor.ApplyMarketData(ctx, item, accountSnapshot)
		if err != nil {
			continue
		}
		actor.handleDecision(actor.botID, decision)
	}
	fatal.OnError(iterator.Err())
}

func (actor *TradeActor) ApplyMarketData(ctx context.Context, message *broker.MarketDataMessage, account tradingstrategy.AccountSnapshot) (tradingstrategy.Decision, error) {
	snapshot := actor.marketState.Apply(message)
	return actor.tradingStrategy.Evaluate(tradingstrategy.NewEvaluateInput(snapshot, account))
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

func (actor *TradeActor) handleDecision(botID string, decision tradingstrategy.Decision) {

}
