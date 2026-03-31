package subscription

import (
	"context"

	"github.com/kduong/trading-backend/internal/eventsource"
)

type Input struct {
	Log    eventsource.Log
	Cursor int64
	Apply  ApplyFunc
}

type ApplyFunc func(ctx context.Context, event *eventsource.Event) error
