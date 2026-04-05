package indicator

import "github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"

// ComputeSMA computes a Simple Moving Average over the given price series.
// Returns one Point per bar starting at index period-1.
func ComputeSMA(prices []replay.PricePoint, period int) []Point {
	if period < 2 || len(prices) < period {
		return nil
	}
	out := make([]Point, 0, len(prices)-period+1)
	sum := 0.0
	for i, p := range prices {
		sum += p.Close
		if i >= period {
			sum -= prices[i-period].Close
		}
		if i < period-1 {
			continue
		}
		out = append(out, Point{At: p.At, Value: sum / float64(period)})
	}
	return out
}
