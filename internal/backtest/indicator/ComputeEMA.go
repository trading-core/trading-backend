package indicator

import "github.com/kduong/trading-backend/internal/backtest/replay"

// ComputeEMA computes an exponential moving average series.
// Seeded with the SMA of the first period bars to avoid initialization bias,
// matching the approach used internally by ComputeMACD.
// The first output point is anchored at prices[period-1].
// Minimum bars required: period.
func ComputeEMA(prices []replay.PricePoint, period int) []Point {
	if len(prices) < period || period < 2 {
		return nil
	}

	k := 2.0 / (float64(period) + 1)

	var sum float64
	for i := 0; i < period; i++ {
		sum += prices[i].Close
	}
	ema := sum / float64(period)

	output := make([]Point, 0, len(prices)-period+1)
	output = append(output, Point{At: prices[period-1].At, Value: ema})

	for i := period; i < len(prices); i++ {
		ema = ema + k*(prices[i].Close-ema)
		output = append(output, Point{At: prices[i].At, Value: ema})
	}

	return output
}
