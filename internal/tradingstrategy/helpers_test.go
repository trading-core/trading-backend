package tradingstrategy_test

import (
	"time"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

type stubStrategy struct {
	decision tradingstrategy.Decision
	calls    int
}

func (s *stubStrategy) Evaluate(input tradingstrategy.EvaluateInput) tradingstrategy.Decision {
	s.calls++
	return s.decision
}

func float64PtrForTest(value float64) *float64 {
	return &value
}

func nyTimeForTest(hour int, minute int) time.Time {
	return time.Date(2026, time.April, 6, hour, minute, 0, 0, tradingstrategy.USMarketLocation)
}
