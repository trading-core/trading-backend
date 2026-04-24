package authz_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/internal/authz"
	"github.com/kduong/trading-backend/internal/contextx"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRequireScope(t *testing.T) {
	Convey("Given a context with scopes", t, func() {
		ctx := contextx.WithScopes(context.Background(), []string{"files:read", "jobs:read"})

		Convey("When the required scope is present, access is allowed", func() {
			err := authz.RequireScope(ctx, "files:read")
			So(err, ShouldBeNil)
		})

		Convey("When the required scope is missing, the sentinel is returned wrapped as 403", func() {
			err := authz.RequireScope(ctx, "files:write")
			So(err, ShouldNotBeNil)
			So(errors.Is(err, authz.ErrScopeDenied), ShouldBeTrue)
			So(merry.HTTPCode(err), ShouldEqual, 403)
		})
	})

	Convey("Given a context with no scopes, every required scope is denied", t, func() {
		err := authz.RequireScope(context.Background(), "files:read")
		So(errors.Is(err, authz.ErrScopeDenied), ShouldBeTrue)
	})
}

func TestRequireOwner(t *testing.T) {
	Convey("Given a context with a caller user", t, func() {
		ctx := contextx.WithUserID(context.Background(), "user-1")

		Convey("When the resource owner matches, access is allowed", func() {
			err := authz.RequireOwner(ctx, "user-1")
			So(err, ShouldBeNil)
		})

		Convey("When the resource belongs to someone else, the sentinel is returned wrapped as 403", func() {
			err := authz.RequireOwner(ctx, "user-2")
			So(err, ShouldNotBeNil)
			So(errors.Is(err, authz.ErrOwnershipDenied), ShouldBeTrue)
			So(merry.HTTPCode(err), ShouldEqual, 403)
		})
	})
}
