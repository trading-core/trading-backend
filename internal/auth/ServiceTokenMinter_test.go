package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kduong/trading-backend/internal/auth"
	. "github.com/smartystreets/goconvey/convey"
)

func TestServiceTokenMinter(t *testing.T) {
	Convey("Given a service token minter", t, func() {
		secret := []byte("test-secret")
		minter := auth.NewServiceTokenMinter(auth.NewServiceTokenMinterInput{
			TokenSecret: secret,
			TTL:         time.Minute,
		})

		validInput := auth.MintTokenInput{
			OnBehalfOfUserID: "user-42",
			Actor:            auth.ActorReportingService,
			Scopes:           []string{"files:read"},
			Audience:         []string{auth.AudienceStorageService},
		}

		Convey("When it mints a token for an acting user", func() {
			signed, err := minter.MintToken(validInput)
			So(err, ShouldBeNil)
			So(signed, ShouldNotBeEmpty)

			Convey("Then the token parses with the expected claims", func() {
				claims := &auth.Claims{}
				parsed, err := jwt.ParseWithClaims(signed, claims, func(*jwt.Token) (interface{}, error) {
					return secret, nil
				})
				So(err, ShouldBeNil)
				So(parsed.Valid, ShouldBeTrue)
				So(claims.Subject, ShouldEqual, "user-42")
				So(claims.Scopes(), ShouldResemble, []string{"files:read"})
				So(claims.Act, ShouldNotBeNil)
				So(claims.Act.Sub, ShouldEqual, auth.ActorReportingService)
				So([]string(claims.Audience), ShouldResemble, []string{auth.AudienceStorageService})
			})
		})

		Convey("When the acting user is missing, minting is rejected", func() {
			input := validInput
			input.OnBehalfOfUserID = ""
			_, err := minter.MintToken(input)
			So(err, ShouldEqual, auth.ErrMissingOnBehalfOfUserID)
		})

		Convey("When the actor is missing, minting is rejected", func() {
			input := validInput
			input.Actor = ""
			_, err := minter.MintToken(input)
			So(err, ShouldEqual, auth.ErrMissingActor)
		})

		Convey("When no scopes are provided, minting is rejected", func() {
			input := validInput
			input.Scopes = nil
			_, err := minter.MintToken(input)
			So(err, ShouldEqual, auth.ErrMissingScopes)
		})

		Convey("When no audience is provided, minting is rejected", func() {
			input := validInput
			input.Audience = nil
			_, err := minter.MintToken(input)
			So(err, ShouldEqual, auth.ErrMissingAudience)
		})

		Convey("When multiple scopes are provided, they round-trip as a space-separated list", func() {
			input := validInput
			input.Scopes = []string{"files:read", "files:write"}
			signed, err := minter.MintToken(input)
			So(err, ShouldBeNil)

			claims := &auth.Claims{}
			_, err = jwt.ParseWithClaims(signed, claims, func(*jwt.Token) (interface{}, error) {
				return secret, nil
			})
			So(err, ShouldBeNil)
			So(claims.Scopes(), ShouldResemble, []string{"files:read", "files:write"})
		})
	})
}
