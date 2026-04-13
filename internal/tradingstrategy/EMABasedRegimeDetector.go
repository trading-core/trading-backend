package tradingstrategy

// EMABasedRegimeDetector classifies the market regime using a fast/slow EMA
// crossover optionally filtered by ADX trend strength and a minimum EMA
// separation threshold (hysteresis).
//
// Classification rules:
//   - FastEMA > SlowEMA and trend is strong → RegimeUptrend
//   - FastEMA < SlowEMA and trend is strong → RegimeDowntrend
//   - Otherwise (low ADX, flat EMAs, or missing data) → RegimeRange
//
// Trend strength check (when ADXThreshold > 0):
//   - ADX must be present and >= ADXThreshold for a trend regime to be declared.
//   - When ADX is unavailable or below threshold, the regime defaults to RegimeRange.
//
// When ADXThreshold is zero, ADX filtering is disabled and regime is determined
// by EMA crossover alone.
//
// EMA separation hysteresis (when EMASeparationThreshold > 0):
//   - A trend regime is only declared when the fractional gap
//     (FastEMA − SlowEMA) / SlowEMA exceeds EMASeparationThreshold.
//   - This prevents rapid regime flipping when the two EMAs are nearly equal
//     at a crossover point.
//   - When EMASeparationThreshold is zero, any crossover qualifies.
//
// When FastEMA or SlowEMA is unavailable, the detector defaults to RegimeRange
// so that downstream range-entry strategies can still fire.
type EMABasedRegimeDetector struct {
	adxThreshold           float64
	emaSeparationThreshold float64
}

type NewEMABasedRegimeDetectorInput struct {
	ADXThreshold           float64 // ADX value at or above which a trend is confirmed; 0 disables ADX filtering
	EMASeparationThreshold float64 // minimum (FastEMA−SlowEMA)/SlowEMA fraction to declare a trend; 0 disables hysteresis
}

func NewEMABasedRegimeDetector(input NewEMABasedRegimeDetectorInput) *EMABasedRegimeDetector {
	return &EMABasedRegimeDetector{
		adxThreshold:           input.ADXThreshold,
		emaSeparationThreshold: input.EMASeparationThreshold,
	}
}

func (detector *EMABasedRegimeDetector) Detect(input EvaluateInput) Regime {
	if input.FastEMA == nil || input.SlowEMA == nil {
		return RegimeRange
	}

	if detector.adxThreshold > 0 {
		if input.ADX == nil || *input.ADX < detector.adxThreshold {
			return RegimeRange
		}
	}

	separation := (*input.FastEMA - *input.SlowEMA) / *input.SlowEMA
	if detector.emaSeparationThreshold > 0 {
		if separation > detector.emaSeparationThreshold {
			return RegimeUptrend
		}
		if separation < -detector.emaSeparationThreshold {
			return RegimeDowntrend
		}
		return RegimeRange
	}

	if *input.FastEMA > *input.SlowEMA {
		return RegimeUptrend
	}
	return RegimeDowntrend
}
