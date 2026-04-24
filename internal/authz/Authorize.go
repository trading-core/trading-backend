// Package authz is the policy enforcement point (PEP) for this repo. Handlers
// consult it — rather than inspecting JWT claims directly — so access rules
// live in one place and can be extended without editing every endpoint.
package authz

import (
	"context"
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/internal/contextx"
)

// Scope identifiers. Tokens advertise these via the OAuth2 `scope` claim and
// handlers require a specific one before servicing a request.
const (
	ScopeFilesRead  = "files:read"
	ScopeFilesWrite = "files:write"
	ScopeJobsRead   = "jobs:read"
	ScopeJobsWrite  = "jobs:write"
)

// UserScopes is the scope set granted to a human user's session token. It
// represents the union of operations a user may perform directly; narrower
// service-minted tokens restrict to a single scope per operation.
var UserScopes = []string{
	ScopeFilesRead,
	ScopeFilesWrite,
	ScopeJobsRead,
	ScopeJobsWrite,
}

// ErrScopeDenied is the sentinel returned when the caller's token does not
// carry the required scope for an operation.
var ErrScopeDenied = errors.New("scope denied")

// ErrOwnershipDenied is the sentinel returned when the caller does not own a
// resource they attempted to access.
var ErrOwnershipDenied = errors.New("ownership denied")

// RequireScope asserts the caller's token carries the given scope. Returns an
// HTTP 403 merry error on denial.
func RequireScope(ctx context.Context, required string) error {
	for _, scope := range contextx.GetScopes(ctx) {
		if scope == required {
			return nil
		}
	}
	return merry.Wrap(ErrScopeDenied).WithHTTPCode(http.StatusForbidden).WithUserMessage("forbidden")
}

// RequireOwner asserts the caller's subject matches the resource owner.
// Returns an HTTP 403 merry error on denial.
func RequireOwner(ctx context.Context, resourceOwnerUserID string) error {
	if contextx.GetUserID(ctx) == resourceOwnerUserID {
		return nil
	}
	return merry.Wrap(ErrOwnershipDenied).WithHTTPCode(http.StatusForbidden).WithUserMessage("forbidden")
}
