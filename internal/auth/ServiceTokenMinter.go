package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kduong/trading-backend/internal/config"
)

// ErrMissingOnBehalfOfUserID is returned when MintToken is called without a user to act on behalf of.
var ErrMissingOnBehalfOfUserID = errors.New("on-behalf-of user id is required")

// ErrMissingActor is returned when MintToken is called without an acting service.
var ErrMissingActor = errors.New("actor is required")

// ErrMissingScopes is returned when MintToken is called without any scopes.
var ErrMissingScopes = errors.New("at least one scope is required")

// ErrMissingAudience is returned when MintToken is called without an audience.
var ErrMissingAudience = errors.New("at least one audience is required")

// ServiceTokenMinter mints short-lived JWTs for internal service-to-service calls.
// Tokens carry the acting user's id in the subject claim, the acting service in
// the act claim (RFC 8693), a narrow scope set, and an audience so downstream
// services can reject tokens that were not minted for them.
type ServiceTokenMinter struct {
	tokenSecret []byte
	ttl         time.Duration
}

type NewServiceTokenMinterInput struct {
	TokenSecret []byte
	TTL         time.Duration
}

func NewServiceTokenMinter(input NewServiceTokenMinterInput) *ServiceTokenMinter {
	return &ServiceTokenMinter{
		tokenSecret: input.TokenSecret,
		ttl:         input.TTL,
	}
}

func ServiceTokenMinterFromEnv() *ServiceTokenMinter {
	return NewServiceTokenMinter(NewServiceTokenMinterInput{
		TokenSecret: []byte(config.EnvStringOrFatal("TOKEN_SECRET")),
		TTL:         config.EnvDuration("SERVICE_TOKEN_TTL", 5*time.Minute),
	})
}

type MintTokenInput struct {
	// OnBehalfOfUserID becomes the subject. Ownership checks downstream compare against this.
	OnBehalfOfUserID string
	// Actor is the service doing the minting (e.g. AudienceReportingService).
	Actor string
	// Scopes is the list of permissions this token grants — narrow to the operation.
	Scopes []string
	// Audience is the list of services authorised to accept this token.
	Audience []string
}

// MintToken returns a signed JWT carrying the acting user, scopes, audience, and actor.
func (minter *ServiceTokenMinter) MintToken(input MintTokenInput) (string, error) {
	if input.OnBehalfOfUserID == "" {
		return "", ErrMissingOnBehalfOfUserID
	}
	if input.Actor == "" {
		return "", ErrMissingActor
	}
	if len(input.Scopes) == 0 {
		return "", ErrMissingScopes
	}
	if len(input.Audience) == 0 {
		return "", ErrMissingAudience
	}
	now := time.Now()
	claims := Claims{
		Scope: strings.Join(input.Scopes, " "),
		Act:   &ActorClaim{Sub: input.Actor},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   input.OnBehalfOfUserID,
			Audience:  jwt.ClaimStrings(input.Audience),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(minter.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(minter.tokenSecret)
}
