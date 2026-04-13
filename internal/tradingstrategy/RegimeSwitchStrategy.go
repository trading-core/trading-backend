package tradingstrategy

// Regime represents a detected market regime.
type Regime int

const (
	RegimeUptrend  Regime = iota
	RegimeRange           // sideways / low-momentum
	RegimeDowntrend
)

// RegimeDetector classifies current market conditions into a Regime
// given the current EvaluateInput.
type RegimeDetector interface {
	Detect(input EvaluateInput) Regime
}

// RegimeSwitchStrategy delegates to one of three sub-strategies based on the
// regime reported by the detector. This allows using distinct entry/exit logic
// for trending vs ranging vs downtrending conditions.
//
// If the detector returns a regime for which no strategy is configured, the
// strategy returns ActionNone.
type RegimeSwitchStrategy struct {
	detector  RegimeDetector
	uptrend   Strategy
	rangeMode Strategy
	downtrend Strategy
}

type NewRegimeSwitchStrategyInput struct {
	Detector  RegimeDetector
	Uptrend   Strategy
	Range     Strategy
	Downtrend Strategy
}

func NewRegimeSwitchStrategy(input NewRegimeSwitchStrategyInput) *RegimeSwitchStrategy {
	return &RegimeSwitchStrategy{
		detector:  input.Detector,
		uptrend:   input.Uptrend,
		rangeMode: input.Range,
		downtrend: input.Downtrend,
	}
}

func (strategy *RegimeSwitchStrategy) Evaluate(input EvaluateInput) Decision {
	regime := strategy.detector.Detect(input)
	switch regime {
	case RegimeUptrend:
		return strategy.uptrend.Evaluate(input)
	case RegimeRange:
		return strategy.rangeMode.Evaluate(input)
	case RegimeDowntrend:
		return strategy.downtrend.Evaluate(input)
	default:
		return Decision{Action: ActionNone, Reason: "unknown regime"}
	}
}
