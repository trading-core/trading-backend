package tradingstrategy

import "time"

type stubStrategy struct {
	typ      StrategyType
	decision Decision
	calls    int
}

func (strategy *stubStrategy) Type() StrategyType {
	return strategy.typ
}

func (strategy *stubStrategy) Evaluate(input EvaluateInput) Decision {
	strategy.calls++
	return strategy.decision
}

func float64PtrForTest(value float64) *float64 {
	return &value
}

func nyTimeForTest(hour int, minute int) time.Time {
	return time.Date(2026, time.April, 6, hour, minute, 0, 0, USMarketLocation)
}
