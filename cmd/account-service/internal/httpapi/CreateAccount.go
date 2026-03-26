package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/account-service/internal/event"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
	uuid "github.com/satori/go.uuid"
)

type CreateAccountInput struct {
	AccountName string `json:"account_name"`
}

type CreateAccountOutput struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
}

func (handler *Handler) CreateAccount(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	// TODO: validate input
	var input CreateAccountInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	accountID := uuid.NewV4().String()
	payload := fatal.UnlessMarshal(event.Frame{
		EventBase: eventsource.NewEventBase(event.EventTypeAccountCreated),
		AccountCreatedEvent: &event.AccountCreatedEvent{
			AccountID:   accountID,
			AccountName: input.AccountName,
			UserID:      contextx.GetUserID(ctx),
		},
	})
	_, err = handler.log.Append(payload)
	fatal.OnError(err)
	output := CreateAccountOutput{
		AccountID:   accountID,
		AccountName: input.AccountName,
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(responseWriter).Encode(&output)
	fatal.OnErrorUnlessDone(ctx, err)
}
