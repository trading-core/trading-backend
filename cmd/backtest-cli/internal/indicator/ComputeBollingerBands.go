package indicator

import (
	"math"

	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"
)

func ComputeBollingerBands(prices []replay.PricePoint, period int, stdDevMultiplier float64) ([]Point, []Point, []Point) {
	if len(prices) < period || period < 2 || stdDevMultiplier <= 0 {
		return nil, nil, nil
	}
	upper := make([]Point, 0, len(prices)-period+1)
	middle := make([]Point, 0, len(prices)-period+1)
	lower := make([]Point, 0, len(prices)-period+1)
	windowSum := 0.0
	windowSqSum := 0.0
	for i := 0; i < len(prices); i++ {
		close := prices[i].Close
		windowSum += close
		windowSqSum += close * close
		if i >= period {
			out := prices[i-period].Close
			windowSum -= out
			windowSqSum -= out * out
		}
		if i < period-1 {
			continue
		}
		mean := windowSum / float64(period)
		variance := (windowSqSum / float64(period)) - (mean * mean)
		if variance < 0 {
			variance = 0
		}
		stddev := math.Sqrt(variance)
		at := prices[i].At
		middle = append(middle, Point{At: at, Value: mean})
		upper = append(upper, Point{At: at, Value: mean + (stdDevMultiplier * stddev)})
		lower = append(lower, Point{At: at, Value: mean - (stdDevMultiplier * stddev)})
	}
	return upper, middle, lower
}
