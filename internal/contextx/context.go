package contextx

import (
	"context"

	"github.com/kduong/trading-backend/internal/fatal"
)

type contextKey string

const (
	contextKeyUserID      contextKey = "user_id"
	contextKeyAccessToken contextKey = "access_token"
	contextKeyScopes      contextKey = "scopes"
	contextKeyActor       contextKey = "actor"
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

// WithScopes stashes the token's scope set on the context. Handlers should
// consult these via the authz package to decide whether the caller may perform
// an action, rather than reaching in directly.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, contextKeyScopes, scopes)
}

// GetScopes returns the scopes attached to the context, or nil if none.
func GetScopes(ctx context.Context) []string {
	v := ctx.Value(contextKeyScopes)
	if v == nil {
		return nil
	}
	scopes, ok := v.([]string)
	if !ok {
		return nil
	}
	return scopes
}

// WithActor stashes the service that minted the token on behalf of the user,
// or an empty string when the token was issued directly to a user.
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, contextKeyActor, actor)
}

// GetActor returns the acting service recorded on the token, or "" if the
// request came directly from a user token.
func GetActor(ctx context.Context) string {
	v := ctx.Value(contextKeyActor)
	if v == nil {
		return ""
	}
	actor, ok := v.(string)
	if !ok {
		return ""
	}
	return actor
}

func WithAccessToken(ctx context.Context, accessToken string) context.Context {
	return context.WithValue(ctx, contextKeyAccessToken, accessToken)
}

func GetAccessToken(ctx context.Context) (accessToken string) {
	v := ctx.Value(contextKeyAccessToken)
	fatal.Unless(v != nil, "access token not found in context")
	accessToken, ok := v.(string)
	fatal.Unless(ok, "access token has wrong type")
	return
}
