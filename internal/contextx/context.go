package contextx

import (
	"context"

	"github.com/kduong/trading-backend/internal/fatal"
)

type contextKey string

const (
	contextKeyAccountID contextKey = "account_id"
)

func WithAccountID(ctx context.Context, accountID string) context.Context {
	return context.WithValue(ctx, contextKeyAccountID, accountID)
}

func GetAccountID(ctx context.Context) (accountID string) {
	v := ctx.Value(contextKeyAccountID)
	fatal.Unless(v != nil, "account ID not found in context")
	accountID, ok := v.(string)
	fatal.Unless(ok, "account ID has wrong type")
	return
}
