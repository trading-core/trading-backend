package indicator

import (
	"math"

	"github.com/kduong/trading-backend/internal/backtest/replay"
)

// ComputeATR computes a close-only ATR series using Wilder's smoothing.
// True range is approximated as |close[t] - close[t-1]| since only close
// prices are available in this data pipeline.
// The first output point is anchored at prices[period] after seeding with
// the simple average of the first N true ranges.
func ComputeATR(prices []replay.PricePoint, period int) []Point {
	if len(prices) <= period || period < 2 {
		return nil
	}
	var sum float64
	for i := 1; i <= period; i++ {
		sum += math.Abs(prices[i].Close - prices[i-1].Close)
	}
	atr := sum / float64(period)
	output := make([]Point, 0, len(prices)-period)
	output = append(output, Point{At: prices[period].At, Value: atr})
	for i := period + 1; i < len(prices); i++ {
		tr := math.Abs(prices[i].Close - prices[i-1].Close)
		atr = ((atr * float64(period-1)) + tr) / float64(period)
		output = append(output, Point{At: prices[i].At, Value: atr})
	}
	return output
}
