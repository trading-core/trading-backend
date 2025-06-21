package bybit

import "context"

type Stream interface {
	Subscribe(ctx context.Context) error
}
