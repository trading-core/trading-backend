package eventsource_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/eventsource"
)

func TestInMemoryLog(t *testing.T) {
	testLog(t, "in-memory", func(channel string) (eventsource.Log, func(), error) {
		return eventsource.NewInMemoryLog(channel), nil, nil
	})
}
