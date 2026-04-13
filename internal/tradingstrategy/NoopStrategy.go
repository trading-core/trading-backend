package tradingstrategy

// NoopStrategy always returns ActionNone, acting as a placeholder in a
// RegimeSwitchStrategy for regimes where no signal should be emitted.
type NoopStrategy struct{}

func NewNoopStrategy() *NoopStrategy {
	return &NoopStrategy{}
}

func (strategy *NoopStrategy) Evaluate(_ EvaluateInput) Decision {
	return Decision{Action: ActionNone, Reason: "noop"}
}
