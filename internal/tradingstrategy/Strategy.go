package tradingstrategy

import (
	"fmt"
	"time"

	"github.com/kduong/trading-backend/internal/fatal"
)

// Regular Trading Hours: 9:30 a.m. – 4:00 p.m. ET (Monday-Friday).
// Pre-Market Session: 4:00 a.m. – 9:30 a.m. ET.
// After-Hours Session: 4:00 p.m. – 8:00 p.m. ET.

var USMarketLocation = loadUSMarketLocation()

func loadUSMarketLocation() *time.Location {
	location, err := time.LoadLocation("America/New_York")
	fatal.OnError(err)
	return location
}

type Action string

const (
	ActionNone Action = "none"
	ActionBuy  Action = "buy"
	ActionSell Action = "sell"
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

func ValidateType(strategyType string) error {
	v := StrategyType(strategyType)
	if _, isValid := ValidStrategyTypes[v]; !isValid {
		return fmt.Errorf("unknown strategy type: %s", strategyType)
	}
	return nil
}

// EvaluateInput is the full decision context passed to a trading strategy.
//
// Price is the preferred executable/reference price derived from the market
// snapshot, while the pointer fields preserve whether specific quote or trade
// values were actually present on the incoming data.
type EvaluateInput struct {
	Price            float64
	LastTradePrice   *float64
	BidPrice         *float64
	AskPrice         *float64
	BidSize          *float64
	AskSize          *float64
	Spread           *float64
	DayVolume        *float64
	LastTradeSize    *float64
	SessionOpenPrice float64
	SessionHighPrice float64
	SessionLowPrice  float64
	CashBalance      float64
	BuyingPower      float64
	PositionQuantity float64
	HasOpenOrder     bool
	EntryPrice       float64
	Now              time.Time
}

type Decision struct {
	Action   Action
	Reason   string
	Quantity float64
}

type Strategy interface {
	Type() StrategyType
	Evaluate(input EvaluateInput) Decision
}

// ScalpingParams allows callers to override default Scalping parameters.
// Zero-valued fields are ignored and defaults are used instead.
type ScalpingParams struct {
	MaxPositionFraction float64
	TakeProfitPct       float64
	SessionStart        int
	SessionEnd          int
}

func New(strategyType string) Strategy {
	return NewWithParams(strategyType, ScalpingParams{})
}

func NewWithParams(strategyType string, params ScalpingParams) Strategy {
	switch strategyType {
	case "scalping":
		s := NewScalping()
		if params.MaxPositionFraction > 0 {
			s.MaxPositionFraction = params.MaxPositionFraction
		}
		if params.TakeProfitPct > 0 {
			s.TakeProfitPct = params.TakeProfitPct
		}
		if params.SessionStart > 0 {
			s.SessionStart = params.SessionStart
		}
		if params.SessionEnd > 0 {
			s.SessionEnd = params.SessionEnd
		}
		return s
	default:
		return new(Noop)
	}
}
