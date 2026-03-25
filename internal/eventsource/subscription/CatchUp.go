package eventsource

import (
	"context"

	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/logger"
)

type CatchUpInput struct {
	Log    eventsource.Log
	Cursor int64
	Apply  func(ctx context.Context, event *eventsource.Event) error
}

func CatchUp(ctx context.Context, input CatchUpInput) (cursor int64, err error) {
	const limit = 1000
	const timeout = 0
	var events []*eventsource.Event
	cursor = input.Cursor
	for {
		events, cursor, err = input.Log.Read(cursor, limit, timeout)
		switch err {
		case nil:
			for _, event := range events {
				if err = input.Apply(ctx, event); err != nil {
					return
				}
			}
		default:
			logger.Fatal(err)
		}
	}
}
