package subscription

import (
	"context"

	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/logger"
)

func CatchUp(ctx context.Context, input Input) (cursor int64, err error) {
	const limit = 1000
	const timeout = 0
	var events []*eventsource.Event
	cursor = input.Cursor
	for {
		events, cursor, err = input.Log.Read(cursor, limit, timeout)
		switch err {
		case nil:
			if len(events) == 0 {
				return
			}
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
