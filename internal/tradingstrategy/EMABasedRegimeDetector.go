package tradingstrategy

// EMABasedRegimeDetector classifies the market regime using a fast/slow EMA
// crossover optionally filtered by ADX trend strength.
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
// When FastEMA or SlowEMA is unavailable, the detector defaults to RegimeRange
// so that downstream range-entry strategies can still fire.
type EMABasedRegimeDetector struct {
	adxThreshold float64
}

type NewEMABasedRegimeDetectorInput struct {
	ADXThreshold float64 // ADX value at or above which a trend is confirmed; 0 disables ADX filtering
}

func NewEMABasedRegimeDetector(input NewEMABasedRegimeDetectorInput) *EMABasedRegimeDetector {
	return &EMABasedRegimeDetector{
		adxThreshold: input.ADXThreshold,
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

	if *input.FastEMA > *input.SlowEMA {
		return RegimeUptrend
	}
	return RegimeDowntrend
}
