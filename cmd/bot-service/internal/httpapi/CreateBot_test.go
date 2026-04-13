package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kduong/trading-backend/cmd/account-service/pkg/accountservice"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/symbolvalidator"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/httpx"
	. "github.com/smartystreets/goconvey/convey"
)

type fakeAccountServiceClient struct {
	account *accountservice.Account
	err     error
}

func (client fakeAccountServiceClient) GetAccount(ctx context.Context, accountID string) (*accountservice.Account, error) {
	return client.account, client.err
}

func (client fakeAccountServiceClient) GetAccountBalance(ctx context.Context, accountID string) (*accountservice.Balance, error) {
	return nil, errors.New("not implemented")
}

type fakeSymbolValidator struct {
	err error
}

func (validator fakeSymbolValidator) Validate(ctx context.Context, brokerType string, symbol string) error {
	return validator.err
}

type fakeBotStoreCommandHandler struct {
	created bool
}

func (handler *fakeBotStoreCommandHandler) Create(ctx context.Context, bot *botstore.Bot) error {
	handler.created = true
	return nil
}

func (handler *fakeBotStoreCommandHandler) UpdateBotStatus(ctx context.Context, botID string, status botstore.BotStatus) error {
	return nil
}

func (handler *fakeBotStoreCommandHandler) Delete(ctx context.Context, botID string) error {
	return nil
}

func TestCreateBotInputValidate(t *testing.T) {
	Convey("Given a create bot input", t, func() {
		Convey("When account and symbol are valid", func() {
			input := CreateBotInput{
				AccountID:         "acct-1",
				Symbol:            "AAPL",
				AllocationPercent: 10,
			}

			err := input.Validate()
			So(err, ShouldBeNil)
		})

		Convey("When symbol has invalid characters", func() {
			input := CreateBotInput{
				AccountID:         "acct-1",
				Symbol:            "BAD!",
				AllocationPercent: 10,
			}

			err := input.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "symbol must be 1-15 chars")
		})
	})
}

func TestCreateBot_RejectsSymbolNotTradableForBroker(t *testing.T) {
	Convey("Given a broker-linked account and non-tradable symbol", t, func() {
		commandHandler := &fakeBotStoreCommandHandler{}
		handler := &Handler{
			accountServiceClient: fakeAccountServiceClient{
				account: &accountservice.Account{
					ID:           "acct-1",
					BrokerLinked: true,
					Broker: &accountservice.BrokerAccount{
						Type: "tastytrade",
						ID:   "broker-1",
					},
				},
			},
			symbolValidator:        fakeSymbolValidator{err: symbolvalidator.ErrSymbolNotTradableForBroker},
			botStoreCommandHandler: commandHandler,
		}

		body := []byte(`{"account_id":"acct-1","symbol":"BADSYMB","allocation_percent":10}`)
		request := httptest.NewRequest(http.MethodPost, "/bots/v1/bots", bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer token")
		request = request.WithContext(contextx.WithUserID(request.Context(), "user-1"))
		recorder := httptest.NewRecorder()

		Convey("When creating a bot", func() {
			handler.CreateBot(recorder, request)

			Convey("Then request is rejected and bot is not created", func() {
				So(recorder.Code, ShouldEqual, http.StatusBadRequest)
				So(commandHandler.created, ShouldBeFalse)
				var message httpx.Message
				err := json.Unmarshal(recorder.Body.Bytes(), &message)
				So(err, ShouldBeNil)
				So(message.Message, ShouldNotBeBlank)
			})
		})
	})
}
