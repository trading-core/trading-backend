package auth

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Audience identifiers — each service that verifies tokens should declare its
// own name so service-to-service tokens can be bound to an intended recipient.
const (
	AudienceAuthenticationService = "authentication-service"
	AudienceStorageService        = "storage-service"
	AudienceReportingService      = "reporting-service"
	AudienceAccountService        = "account-service"
	AudienceBotService            = "bot-service"
	AudienceJournalService        = "journal-service"
	AudienceStockScreenerService  = "stock-screener"
)

// Service identifiers — used as the `act.sub` (actor) value on service-minted
// tokens so audit logs can distinguish a user calling directly from a service
// proxying on their behalf.
const (
	ActorReportingService = "reporting-service"
)

// Claims are the JWT claims carried by tokens issued in this system.
type Claims struct {
	// Scope is an OAuth2-style space-separated list of permissions.
	Scope string `json:"scope,omitempty"`
	// Act carries the acting principal (RFC 8693) when a service mints a token
	// on behalf of a user. It is nil on direct user tokens.
	Act *ActorClaim `json:"act,omitempty"`
	jwt.RegisteredClaims
}

// ActorClaim identifies the service acting on behalf of the subject.
type ActorClaim struct {
	Sub string `json:"sub"`
}

// Scopes splits the space-separated Scope field into individual scope strings.
func (claims *Claims) Scopes() []string {
	if claims.Scope == "" {
		return nil
	}
	return strings.Fields(claims.Scope)
}
