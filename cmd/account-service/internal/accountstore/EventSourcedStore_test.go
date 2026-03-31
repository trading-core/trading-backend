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
		var commandHandler accountstore.CommandHandler
		commandHandler = accountstore.NewEventSourcedCommandHandler(accountstore.NewEventSourcedCommandHandlerInput{
			Log: log,
		})
		var queryHandler accountstore.QueryHandler
		queryHandler = accountstore.NewEventSourcedQueryHandler(accountstore.NewEventSourcedQueryHandlerInput{
			Log: log,
		})
		Convey("When creating multiple accounts for a user", func() {
			userCtx := contextx.WithUserID(context.Background(), "user-1")
			err1 := commandHandler.Create(userCtx, accountstore.CreateInput{
				AccountID:   "account-1",
				AccountName: "Primary",
			})
			err2 := commandHandler.Create(userCtx, accountstore.CreateInput{
				AccountID:   "account-2",
				AccountName: "Secondary",
			})
			Convey("Then both accounts are created successfully", func() {
				So(err1, ShouldBeNil)
				So(err2, ShouldBeNil)
			})
			Convey("And both appear in owner's list", func() {
				listed, err := queryHandler.List(userCtx)
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
				acc, err := queryHandler.Get(userCtx, accountstore.GetInput{AccountID: "account-1"})
				So(err, ShouldBeNil)
				So(acc.Name, ShouldEqual, "Primary")
			})
			Convey("And different user cannot see these accounts", func() {
				otherCtx := contextx.WithUserID(context.Background(), "user-2")
				listed, err := queryHandler.List(otherCtx)
				So(err, ShouldBeNil)
				So(len(listed), ShouldEqual, 0)

				acc, err := queryHandler.Get(otherCtx, accountstore.GetInput{AccountID: "account-1"})
				So(err, ShouldEqual, accountstore.ErrAccountForbidden)
				So(acc, ShouldBeNil)
			})
			Convey("When linking broker account", func() {
				brokerAcc := &broker.Account{
					Type: broker.AccountTypeTastyTrade,
					ID:   "tastytrade-123",
				}
				err := commandHandler.LinkBrokerAccount(userCtx, accountstore.LinkBrokerAccountInput{
					AccountID:     "account-1",
					BrokerAccount: brokerAcc,
				})
				Convey("Then link succeeds", func() {
					So(err, ShouldBeNil)
				})
				Convey("And account shows as linked", func() {
					acc, err := queryHandler.Get(userCtx, accountstore.GetInput{AccountID: "account-1"})
					So(err, ShouldBeNil)
					So(acc.BrokerLinked, ShouldBeTrue)
					So(acc.BrokerAccount.ID, ShouldEqual, "tastytrade-123")
					So(acc.BrokerAccount.Type, ShouldEqual, broker.AccountTypeTastyTrade)
				})
				Convey("And cannot link again", func() {
					err := commandHandler.LinkBrokerAccount(userCtx, accountstore.LinkBrokerAccountInput{
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
				err := commandHandler.LinkBrokerAccount(otherCtx, accountstore.LinkBrokerAccountInput{
					AccountID:     "account-1",
					BrokerAccount: brokerAcc,
				})
				Convey("Then it fails with ErrAccountForbidden", func() {
					So(err, ShouldEqual, accountstore.ErrAccountForbidden)
				})
			})
			Convey("When trying to link to nonexistent account", func() {
				brokerAcc := &broker.Account{
					Type: broker.AccountTypeTastyTrade,
					ID:   "tastytrade-789",
				}
				err := commandHandler.LinkBrokerAccount(userCtx, accountstore.LinkBrokerAccountInput{
					AccountID:     "nonexistent",
					BrokerAccount: brokerAcc,
				})
				Convey("Then it fails with ErrAccountNotFound", func() {
					So(err, ShouldEqual, accountstore.ErrAccountNotFound)
				})
			})
		})
	})
}
