package botsync

import (
	"context"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"
	"github.com/kduong/trading-backend/internal/broker"
)

type Actor struct {
	TradingStrategy tradingstrategy.Strategy
	marketState     *MarketState
}

type NewTradeActorInput struct {
	TradingStrategy tradingstrategy.Strategy
	Symbol          string
}

func NewActor(input NewTradeActorInput) *Actor {
	return &Actor{
		TradingStrategy: input.TradingStrategy,
		marketState:     NewMarketState(input.Symbol),
	}
}

func (actor *Actor) ApplyMarketData(ctx context.Context, message *broker.MarketDataMessage, account tradingstrategy.AccountSnapshot) (tradingstrategy.Decision, error) {
	_ = ctx
	snapshot := actor.marketState.Apply(message)
	return actor.TradingStrategy.Evaluate(tradingstrategy.NewEvaluateInput(snapshot, account))
}
