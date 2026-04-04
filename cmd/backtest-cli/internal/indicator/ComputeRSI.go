package indicator

import (
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"
)

func ComputeRSI(prices []replay.PricePoint, period int) []Point {
	if len(prices) <= period || period < 2 {
		return nil
	}
	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		delta := prices[i].Close - prices[i-1].Close
		if delta > 0 {
			gainSum += delta
		} else {
			lossSum -= delta
		}
	}
	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)
	out := make([]Point, 0, len(prices)-period)
	out = append(out, Point{At: prices[period].At, Value: rsiFromAverages(avgGain, avgLoss)})
	for i := period + 1; i < len(prices); i++ {
		delta := prices[i].Close - prices[i-1].Close
		gain := 0.0
		loss := 0.0
		if delta > 0 {
			gain = delta
		} else {
			loss = -delta
		}
		avgGain = ((avgGain * float64(period-1)) + gain) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + loss) / float64(period)
		out = append(out, Point{At: prices[i].At, Value: rsiFromAverages(avgGain, avgLoss)})
	}
	return out
}

func rsiFromAverages(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}
