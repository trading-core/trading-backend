package contextx

import (
	"context"

	"github.com/kduong/trading-backend/internal/fatal"
)

type contextKey string

const (
	contextKeyFooBar contextKey = "context_key"
)

func WithFooBar(ctx context.Context, fooBar string) context.Context {
	return context.WithValue(ctx, contextKeyFooBar, fooBar)
}

func GetFooBar(ctx context.Context) (fooBar string) {
	v := ctx.Value(contextKeyFooBar)
	fatal.Unless(v != nil, "foo bar not found in context")
	fooBar, ok := v.(string)
	fatal.Unless(ok, "foo bar has wrong type")
	return
}
