package tradingstrategy

import (
	"time"
	_ "time/tzdata"

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
	Evaluate(input EvaluateInput) Decision
}

// Parameters allows callers to override default Scalping parameters.
// Zero-valued fields are ignored and defaults are used instead.
type Parameters struct {
	EntryMode                string  `json:"entry_mode,omitempty"`
	MaxPositionFraction      float64 `json:"max_position_fraction,omitempty"`
	TakeProfitPct            float64 `json:"take_profit_pct,omitempty"`
	StopLossPct              float64 `json:"stop_loss_pct,omitempty"`
	SessionStart             int     `json:"session_start,omitempty"`
	SessionEnd               int     `json:"session_end,omitempty"`
	MinRSI                   float64 `json:"min_rsi,omitempty"`
	RequireMACDSignal        bool    `json:"require_macd_signal,omitempty"`
	RequireBollingerBreakout bool    `json:"require_bollinger_breakout,omitempty"`
	MinBollingerWidthPct     float64 `json:"min_bollinger_width_pct,omitempty"`
	RequireBollingerSqueeze  bool    `json:"require_bollinger_squeeze,omitempty"`
	MaxBollingerWidthPct     float64 `json:"max_bollinger_width_pct,omitempty"`
	ReentryCooldownMinutes   int     `json:"reentry_cooldown_minutes,omitempty"`
	UseVolatilityTP          bool    `json:"use_volatility_tp,omitempty"`
	VolatilityTPMultiplier   float64 `json:"volatility_tp_multiplier,omitempty"`
	RiskPerTradePct          float64 `json:"risk_per_trade_pct,omitempty"`
	BreakoutLookbackBars     int     `json:"breakout_lookback_bars,omitempty"` // number of bars to lookback for breakout (1=session high, 5=5-bar high). Default 1.
}

func FromParameters(parameters *Parameters) Strategy {
	s := NewScalping()
	if parameters.EntryMode != "" {
		s.EntryMode = parameters.EntryMode
	}
	if parameters.MaxPositionFraction > 0 {
		s.MaxPositionFraction = parameters.MaxPositionFraction
	}
	if parameters.TakeProfitPct > 0 {
		s.TakeProfitPct = parameters.TakeProfitPct
	}
	if parameters.SessionStart > 0 {
		s.SessionStart = parameters.SessionStart
	}
	if parameters.SessionEnd > 0 {
		s.SessionEnd = parameters.SessionEnd
	}
	if parameters.MinRSI > 0 {
		s.MinRSI = parameters.MinRSI
	}
	if parameters.StopLossPct > 0 {
		s.StopLossPct = parameters.StopLossPct
	}
	s.RequireMACDSignal = parameters.RequireMACDSignal
	s.RequireBollingerBreakout = parameters.RequireBollingerBreakout
	if parameters.MinBollingerWidthPct > 0 {
		s.MinBollingerWidthPct = parameters.MinBollingerWidthPct
	}
	s.RequireBollingerSqueeze = parameters.RequireBollingerSqueeze
	if parameters.MaxBollingerWidthPct > 0 {
		s.MaxBollingerWidthPct = parameters.MaxBollingerWidthPct
	}
	if parameters.ReentryCooldownMinutes > 0 {
		s.ReentryCooldownMinutes = parameters.ReentryCooldownMinutes
	}
	s.UseVolatilityTP = parameters.UseVolatilityTP
	if parameters.VolatilityTPMultiplier > 0 {
		s.VolatilityTPMultiplier = parameters.VolatilityTPMultiplier
	}
	if parameters.RiskPerTradePct > 0 {
		s.RiskPerTradePct = parameters.RiskPerTradePct
	}
	if parameters.BreakoutLookbackBars > 0 {
		s.BreakoutLookbackBars = parameters.BreakoutLookbackBars
	}
	return s
}
