package subscription

import (
	"context"
	"time"

	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/logger"
)

func Live(ctx context.Context, input Input) (cursor int64, err error) {
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
			err = nil
			return
		default:
			logger.Fatal(err)
		}
	}
}
