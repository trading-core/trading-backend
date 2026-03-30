package subscription

import (
	"context"
	"time"

	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/logger"
)

type LiveInput struct {
	Log    eventsource.Log
	Cursor int64
	Apply  func(ctx context.Context, event *eventsource.Event) error
}

func Live(ctx context.Context, input LiveInput) (cursor int64, err error) {
	const limit = 1000
	const timeout = 10 * time.Second
	var events []*eventsource.Event
	cursor = input.Cursor
	for {
		events, cursor, err = input.Log.Read(cursor, limit, timeout.Milliseconds())
		switch err {
		case nil:
			for _, event := range events {
				if err = input.Apply(ctx, event); err != nil {
					return
				}
			}
		case eventsource.Timeout:
			return
		default:
			logger.Fatal(err)
		}
	}
}
