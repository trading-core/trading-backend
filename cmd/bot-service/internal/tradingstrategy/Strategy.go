package tradingstrategy

import (
	"errors"
	"fmt"
	"time"
)

var ErrUnknownStrategyType = errors.New("unknown strategy type")

type Action string

const (
	ActionNone Action = "none"
	ActionBuy  Action = "buy"
	ActionSell Action = "sell"
	ActionExit Action = "exit"
)

type StrategyType string

const (
	StrategyTypeTrendTrading    StrategyType = "trend_trading"
	StrategyTypeSwingTrading    StrategyType = "swing_trading"
	StrategyTypeScalping        StrategyType = "scalping"
	StrategyTypeBreakoutTrading StrategyType = "breakout_trading"
)

var ValidStrategyTypes = map[StrategyType]struct{}{
	StrategyTypeTrendTrading:    {},
	StrategyTypeSwingTrading:    {},
	StrategyTypeScalping:        {},
	StrategyTypeBreakoutTrading: {},
}

func Validate(strategy Strategy) error {
	strategyType := strategy.Type()
	if _, isValid := ValidStrategyTypes[strategyType]; !isValid {
		return fmt.Errorf("%w: %s", ErrUnknownStrategyType, strategyType)
	}
	return nil
}

type EvaluateInput struct {
	Price            float64
	SessionOpenPrice float64
	SessionHighPrice float64
	SessionLowPrice  float64
	CashBalance      float64
	BuyingPower      float64
	PositionQuantity float64
	HasOpenOrder     bool
	Now              time.Time
}

type Decision struct {
	Action   Action
	Reason   string
	Quantity float64
}

type Strategy interface {
	Type() StrategyType
	Evaluate(input EvaluateInput) (Decision, error)
}

func New(strategyType string) Strategy {
	switch strategyType {
	case "scalping":
		return NewScalping(defaultScalpingConfig)
	default:
		return new(Unknown)
	}
}
