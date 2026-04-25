package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httpx"
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
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	// TODO: validate input
	input, err := httpx.DecodeJSONBody[CreateAccountInput](request)
	if err != nil {
		return
	}
	accountID := uuid.NewV4().String()
	err = handler.accountStoreCommandHandler.Create(ctx, accountstore.CreateInput{
		AccountID:   accountID,
		AccountName: input.AccountName,
	})
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
