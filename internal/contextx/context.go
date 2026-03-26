package contextx

import (
	"context"

	"github.com/kduong/trading-backend/internal/fatal"
)

type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
)

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

func GetUserID(ctx context.Context) (userID string) {
	v := ctx.Value(contextKeyUserID)
	fatal.Unless(v != nil, "user ID not found in context")
	userID, ok := v.(string)
	fatal.Unless(ok, "user ID has wrong type")
	return
}
