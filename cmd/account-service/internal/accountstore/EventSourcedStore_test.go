package accountstore_test

import (
	"context"
	"testing"

	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEventSourcedStoreTest(t *testing.T) {
	Convey("Given an event sourced store with multiple accounts", t, func() {
		log := eventsource.NewInMemoryLog("accounts")
		store := accountstore.NewEventSourcedStore(accountstore.NewEventSourcedStoreInput{Log: log})
		Convey("When creating multiple accounts for a user", func() {
			userCtx := contextx.WithUserID(context.Background(), "user-1")
			err1 := store.Create(userCtx, accountstore.CreateInput{
				AccountID:   "account-1",
				AccountName: "Primary",
			})
			err2 := store.Create(userCtx, accountstore.CreateInput{
				AccountID:   "account-2",
				AccountName: "Secondary",
			})
			Convey("Then both accounts are created successfully", func() {
				So(err1, ShouldBeNil)
				So(err2, ShouldBeNil)
			})
			Convey("And both appear in owner's list", func() {
				listed, err := store.List(userCtx)
				So(err, ShouldBeNil)
				So(len(listed), ShouldEqual, 2)
				idMap := make(map[string]string)
				for _, acc := range listed {
					idMap[acc.ID] = acc.Name
				}
				So(idMap["account-1"], ShouldEqual, "Primary")
				So(idMap["account-2"], ShouldEqual, "Secondary")
			})
			Convey("And user can get specific account", func() {
				acc, err := store.Get(userCtx, accountstore.GetInput{AccountID: "account-1"})
				So(err, ShouldBeNil)
				So(acc.Name, ShouldEqual, "Primary")
			})
			Convey("And different user cannot see these accounts", func() {
				otherCtx := contextx.WithUserID(context.Background(), "user-2")
				listed, err := store.List(otherCtx)
				So(err, ShouldBeNil)
				So(len(listed), ShouldEqual, 0)

				acc, err := store.Get(otherCtx, accountstore.GetInput{AccountID: "account-1"})
				So(err, ShouldEqual, accountstore.ErrForbidden)
				So(acc, ShouldBeNil)
			})
			Convey("When linking broker account", func() {
				brokerAcc := &broker.Account{
					Type: broker.AccountTypeTastyTrade,
					ID:   "tastytrade-123",
				}
				err := store.LinkBrokerAccount(userCtx, accountstore.LinkBrokerAccountInput{
					AccountID:     "account-1",
					BrokerAccount: brokerAcc,
				})
				Convey("Then link succeeds", func() {
					So(err, ShouldBeNil)
				})
				Convey("And account shows as linked", func() {
					acc, err := store.Get(userCtx, accountstore.GetInput{AccountID: "account-1"})
					So(err, ShouldBeNil)
					So(acc.BrokerLinked, ShouldBeTrue)
					So(acc.BrokerAccount.ID, ShouldEqual, "tastytrade-123")
					So(acc.BrokerAccount.Type, ShouldEqual, broker.AccountTypeTastyTrade)
				})
				Convey("And cannot link again", func() {
					err := store.LinkBrokerAccount(userCtx, accountstore.LinkBrokerAccountInput{
						AccountID:     "account-1",
						BrokerAccount: brokerAcc,
					})
					So(err, ShouldEqual, accountstore.ErrBrokerAccountAlreadyLinked)
				})
			})
			Convey("When other user tries to link broker account", func() {
				otherCtx := contextx.WithUserID(context.Background(), "user-2")
				brokerAcc := &broker.Account{
					Type: broker.AccountTypeTastyTrade,
					ID:   "tastytrade-456",
				}
				err := store.LinkBrokerAccount(otherCtx, accountstore.LinkBrokerAccountInput{
					AccountID:     "account-1",
					BrokerAccount: brokerAcc,
				})
				Convey("Then it fails with ErrForbidden", func() {
					So(err, ShouldEqual, accountstore.ErrForbidden)
				})
			})
			Convey("When trying to link to nonexistent account", func() {
				brokerAcc := &broker.Account{
					Type: broker.AccountTypeTastyTrade,
					ID:   "tastytrade-789",
				}
				err := store.LinkBrokerAccount(userCtx, accountstore.LinkBrokerAccountInput{
					AccountID:     "nonexistent",
					BrokerAccount: brokerAcc,
				})
				Convey("Then it fails with ErrNotFound", func() {
					So(err, ShouldEqual, accountstore.ErrNotFound)
				})
			})
		})
	})
}
