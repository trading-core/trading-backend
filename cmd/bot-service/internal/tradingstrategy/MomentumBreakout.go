package tradingstrategy

import (
	"context"
	"fmt"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
)

type MomentumBreakout struct{}

func (MomentumBreakout) Type() string {
	return "momentum_breakout"
}

func (strategy MomentumBreakout) Validate(bot *botstore.Bot) error {
	if bot == nil {
		return fmt.Errorf("bot is required")
	}
	if bot.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if bot.AllocationPercent <= 0 {
		return fmt.Errorf("allocation_percent must be greater than 0")
	}
	if bot.StrategyTradeType != strategy.Type() {
		return fmt.Errorf("invalid strategy type %q for %s", bot.StrategyTradeType, strategy.Type())
	}
	return nil
}

func (MomentumBreakout) Evaluate(ctx context.Context, input EvaluateInput) (Decision, error) {
	_ = ctx
	if input.Bot == nil {
		return Decision{}, fmt.Errorf("bot is required")
	}
	if input.HasOpenOrder {
		return Decision{Action: ActionNone, Reason: "waiting for open order to resolve"}, nil
	}
	if input.Price <= 0 {
		return Decision{Action: ActionNone, Reason: "price unavailable"}, nil
	}
	if input.PositionQuantity <= 0 && input.CashBalance <= 0 {
		return Decision{Action: ActionNone, Reason: "no cash available"}, nil
	}

	// Minimal sample rules so the runner has a deterministic contract:
	// buy on breakout above the session high, exit if price loses the session open.
	if input.PositionQuantity <= 0 && input.Price > input.SessionHighPrice && input.CashBalance > 0 {
		return Decision{
			Action:   ActionBuy,
			Reason:   "price broke above session high",
			Quantity: 1,
		}, nil
	}
	if input.PositionQuantity > 0 && input.Price < input.SessionOpenPrice {
		return Decision{
			Action:   ActionExit,
			Reason:   "price lost session open",
			Quantity: input.PositionQuantity,
		}, nil
	}

	return Decision{Action: ActionNone, Reason: "no momentum breakout signal"}, nil
}
