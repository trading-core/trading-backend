package indicator

import "github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"

func ComputeMACD(prices []replay.PricePoint, fastPeriod int, slowPeriod int, signalPeriod int) ([]Point, []Point) {
	if len(prices) == 0 || fastPeriod < 2 || slowPeriod < 2 || signalPeriod < 2 || slowPeriod <= fastPeriod {
		return nil, nil
	}
	fastK := 2.0 / (float64(fastPeriod) + 1)
	slowK := 2.0 / (float64(slowPeriod) + 1)
	signalK := 2.0 / (float64(signalPeriod) + 1)
	fastEMA := prices[0].Close
	slowEMA := prices[0].Close
	macdSeries := make([]Point, 0, len(prices))
	signalSeries := make([]Point, 0, len(prices))
	var signalEMA float64
	hasSignal := false
	for i, p := range prices {
		if i > 0 {
			fastEMA = ((p.Close - fastEMA) * fastK) + fastEMA
			slowEMA = ((p.Close - slowEMA) * slowK) + slowEMA
		}
		macd := fastEMA - slowEMA
		macdSeries = append(macdSeries, Point{At: p.At, Value: macd})
		if !hasSignal {
			signalEMA = macd
			hasSignal = true
		} else {
			signalEMA = ((macd - signalEMA) * signalK) + signalEMA
		}
		signalSeries = append(signalSeries, Point{At: p.At, Value: signalEMA})
	}
	return macdSeries, signalSeries
}
