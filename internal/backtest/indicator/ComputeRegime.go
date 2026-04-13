package indicator

// Regime values returned by ComputeRegime.
const (
	RegimeUptrend   = 0.0
	RegimeRange     = 1.0
	RegimeDowntrend = 2.0
)

// ComputeRegime classifies the market regime at each bar where both fastEMA
// and slowEMA are available. When adxThreshold > 0, ADX must be present and
// at or above the threshold for a trend regime to be declared; otherwise the
// regime defaults to RegimeRange.
//
// Classification rules:
//   - FastEMA > SlowEMA and trend is confirmed → RegimeUptrend (0)
//   - FastEMA < SlowEMA and trend is confirmed → RegimeDowntrend (2)
//   - Otherwise (low ADX, flat EMAs, missing data) → RegimeRange (1)
//
// The returned series has one point per bar where fastEMA and slowEMA overlap
// by timestamp. Point.Value encodes the regime as 0, 1, or 2.
func ComputeRegime(fastEMA, slowEMA, adx []Point, adxThreshold float64) []Point {
	slowByTime := make(map[int64]float64, len(slowEMA))
	for _, p := range slowEMA {
		slowByTime[p.At.Unix()] = p.Value
	}
	adxByTime := make(map[int64]float64, len(adx))
	for _, p := range adx {
		adxByTime[p.At.Unix()] = p.Value
	}

	out := make([]Point, 0, len(fastEMA))
	for _, fast := range fastEMA {
		ts := fast.At.Unix()
		slow, hasSlow := slowByTime[ts]
		if !hasSlow {
			continue
		}
		regime := RegimeRange
		if adxThreshold > 0 {
			adxValue, hasADX := adxByTime[ts]
			if hasADX && adxValue >= adxThreshold {
				if fast.Value > slow {
					regime = RegimeUptrend
				} else {
					regime = RegimeDowntrend
				}
			}
		} else {
			if fast.Value > slow {
				regime = RegimeUptrend
			} else {
				regime = RegimeDowntrend
			}
		}
		out = append(out, Point{At: fast.At, Value: regime})
	}
	return out
}
