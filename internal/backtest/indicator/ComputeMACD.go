package indicator

import "github.com/kduong/trading-backend/internal/backtest/replay"

// ComputeMACD computes the MACD line and signal line.
//
// Both EMAs are seeded with their respective SMAs to avoid initialization bias
// (starting both at prices[0] causes the MACD to appear flat for the first
// slowPeriod bars while the EMAs converge at different rates).
//
// MACD values are emitted from bar slowPeriod onward (once both EMAs are seeded).
// Signal values are emitted from MACD index signalPeriod onward (once the
// signal EMA is seeded with its own SMA).
//
// Minimum bars required: slowPeriod + signalPeriod.
func ComputeMACD(prices []replay.PricePoint, fastPeriod int, slowPeriod int, signalPeriod int) ([]Point, []Point) {
	if len(prices) < slowPeriod+signalPeriod || fastPeriod < 2 || slowPeriod < 2 || signalPeriod < 2 || slowPeriod <= fastPeriod {
		return nil, nil
	}

	fastK := 2.0 / (float64(fastPeriod) + 1)
	slowK := 2.0 / (float64(slowPeriod) + 1)
	signalK := 2.0 / (float64(signalPeriod) + 1)

	// Seed fast EMA with SMA of its first fastPeriod bars.
	var fastSum float64
	for i := 0; i < fastPeriod; i++ {
		fastSum += prices[i].Close
	}
	fastEMA := fastSum / float64(fastPeriod)

	// Seed slow EMA with SMA of its first slowPeriod bars.
	var slowSum float64
	for i := 0; i < slowPeriod; i++ {
		slowSum += prices[i].Close
	}
	slowEMA := slowSum / float64(slowPeriod)

	// Advance fast EMA from bar fastPeriod through slowPeriod-1 so both
	// EMAs are aligned and ready to process bar slowPeriod together.
	for i := fastPeriod; i < slowPeriod; i++ {
		fastEMA = fastEMA + fastK*(prices[i].Close-fastEMA)
	}

	// Compute MACD from bar slowPeriod onward (both EMAs fully seeded).
	macdPoints := make([]Point, 0, len(prices)-slowPeriod)
	for i := slowPeriod; i < len(prices); i++ {
		fastEMA = fastEMA + fastK*(prices[i].Close-fastEMA)
		slowEMA = slowEMA + slowK*(prices[i].Close-slowEMA)
		macdPoints = append(macdPoints, Point{At: prices[i].At, Value: fastEMA - slowEMA})
	}

	// Seed signal EMA with SMA of first signalPeriod MACD values.
	var signalSum float64
	for i := 0; i < signalPeriod; i++ {
		signalSum += macdPoints[i].Value
	}
	signalEMA := signalSum / float64(signalPeriod)

	// Signal EMA from MACD index signalPeriod onward.
	signalPoints := make([]Point, 0, len(macdPoints)-signalPeriod)
	for i := signalPeriod; i < len(macdPoints); i++ {
		signalEMA = signalEMA + signalK*(macdPoints[i].Value-signalEMA)
		signalPoints = append(signalPoints, Point{At: macdPoints[i].At, Value: signalEMA})
	}

	return macdPoints, signalPoints
}
