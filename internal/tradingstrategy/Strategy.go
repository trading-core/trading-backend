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
	StrategyTypeScalping        StrategyType = "scalping"
	StrategyTypePullbackTrading StrategyType = "pullback_trading"
	StrategyTypeBreakoutTrading StrategyType = "breakout_trading"
)

var ValidStrategyTypes = map[StrategyType]struct{}{
	StrategyTypeScalping:        {},
	StrategyTypePullbackTrading: {},
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
	Price             float64
	RSI               *float64
	MACD              *float64
	MACDSignal        *float64
	BollUpper         *float64
	BollMiddle        *float64
	BollLower         *float64
	BollWidthPct      *float64
	LastTradePrice    *float64
	BidPrice          *float64
	AskPrice          *float64
	BidSize           *float64
	AskSize           *float64
	Spread            *float64
	DayVolume         *float64
	LastTradeSize     *float64
	SessionOpenPrice  float64
	SessionHighPrice  float64
	SessionLowPrice   float64
	LookbackHighPrice float64 // N-bar high for longer timeframes (e.g., 5-day high)
	LookbackLowPrice  float64 // N-bar low for longer timeframes
	CashBalance       float64
	BuyingPower       float64
	PositionQuantity  float64
	HasOpenOrder      bool
	EntryPrice        float64
	HighSinceEntry    float64
	LastStopLossAt    *time.Time
	Now               time.Time
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
	EntryMode                string
	MaxPositionFraction      float64
	TakeProfitPct            float64
	StopLossPct              float64
	SessionStart             int
	SessionEnd               int
	MinRSI                   float64
	RequireMACDSignal        bool
	RequireBollingerBreakout bool
	MinBollingerWidthPct     float64
	RequireBollingerSqueeze  bool
	MaxBollingerWidthPct     float64
	ReentryCooldownMinutes   int
	UseVolatilityTP          bool
	VolatilityTPMultiplier   float64
	RiskPerTradePct          float64
	BreakoutLookbackBars     int // number of bars to lookback for breakout (1=session high, 5=5-bar high). Default 1.
}

func New(strategyType string) Strategy {
	return NewWithParams(strategyType, ScalpingParams{})
}

func NewWithParams(strategyType string, params ScalpingParams) Strategy {
	switch strategyType {
	case "scalping":
		s := NewScalping()
		if params.EntryMode != "" {
			s.EntryMode = params.EntryMode
		}
		if params.MaxPositionFraction > 0 {
			s.MaxPositionFraction = params.MaxPositionFraction
		}
		if params.TakeProfitPct > 0 {
			s.TakeProfitPct = params.TakeProfitPct
		}
		if params.SessionStart >= 0 {
			s.SessionStart = params.SessionStart
		}
		if params.SessionEnd > 0 {
			s.SessionEnd = params.SessionEnd
		}
		if params.MinRSI > 0 {
			s.MinRSI = params.MinRSI
		}
		if params.StopLossPct > 0 {
			s.StopLossPct = params.StopLossPct
		}
		s.RequireMACDSignal = params.RequireMACDSignal
		s.RequireBollingerBreakout = params.RequireBollingerBreakout
		if params.MinBollingerWidthPct > 0 {
			s.MinBollingerWidthPct = params.MinBollingerWidthPct
		}
		s.RequireBollingerSqueeze = params.RequireBollingerSqueeze
		if params.MaxBollingerWidthPct > 0 {
			s.MaxBollingerWidthPct = params.MaxBollingerWidthPct
		}
		if params.ReentryCooldownMinutes > 0 {
			s.ReentryCooldownMinutes = params.ReentryCooldownMinutes
		}
		s.UseVolatilityTP = params.UseVolatilityTP
		if params.VolatilityTPMultiplier > 0 {
			s.VolatilityTPMultiplier = params.VolatilityTPMultiplier
		}
		if params.RiskPerTradePct > 0 {
			s.RiskPerTradePct = params.RiskPerTradePct
		}
		if params.BreakoutLookbackBars > 0 {
			s.BreakoutLookbackBars = params.BreakoutLookbackBars
		}
		return s
	default:
		return nil
	}
}
