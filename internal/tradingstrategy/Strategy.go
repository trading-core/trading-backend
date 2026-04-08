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
	// ActionVeto is a hard block emitted by guard strategies (session, system, balance).
	// It propagates through a CompositeStrategy and overrides any vote result,
	// unlike ActionNone which is a signal absence that participates in voting.
	ActionVeto Action = "veto"
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
	SMA               *float64
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
	LastStopLossAt       *time.Time
	LastOverboughtExitAt *time.Time
	MACDAboveSinceEntry  bool
	Now                  time.Time
}

type Decision struct {
	Action   Action  `json:"action"`
	Reason   string  `json:"reason"`
	Quantity float64 `json:"quantity"`
}

type Strategy interface {
	Evaluate(input EvaluateInput) Decision
}
