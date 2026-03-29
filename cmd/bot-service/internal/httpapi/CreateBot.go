package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/account-service/pkg/accountservice"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
	"github.com/kduong/trading-backend/internal/logger"
	uuid "github.com/satori/go.uuid"
)

type CreateBotInput struct {
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
}

func (handler *Handler) CreateBot(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	var input CreateBotInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest)
		return
	}
	if input.AccountID == "" || input.Name == "" {
		err = merry.New("account_id and name are required").WithHTTPCode(http.StatusBadRequest)
		return
	}
	authorization := request.Header.Get("Authorization")
	parts := strings.SplitN(authorization, " ", 2)
	fatal.Unless(len(parts) == 2, "invalid authorization header format")
	ctx = contextx.WithAccessToken(ctx, parts[1])
	account, err := handler.accountServiceClient.GetAccount(ctx, input.AccountID)
	switch {
	case err == nil:
	case errors.Is(err, accountservice.ErrAccountNotFound):
		err = merry.Wrap(err).WithHTTPCode(http.StatusNotFound)
		return
	case errors.Is(err, accountservice.ErrAccountForbidden):
		err = merry.Wrap(err).WithHTTPCode(http.StatusForbidden)
		return
	case errors.Is(err, accountservice.ErrServerError):
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadGateway)
		return
	default:
		logger.Fatal(err)
	}
	if !account.BrokerLinked {
		err = merry.New("account is not linked to a broker").WithHTTPCode(http.StatusBadRequest)
		return
	}
	bot := &botstore.Bot{
		ID:              uuid.NewV4().String(),
		UserID:          userID,
		AccountID:       account.ID,
		BrokerAccountID: account.Broker.ID,
		BrokerType:      account.Broker.Type,
		Name:            input.Name,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	err = handler.botStore.Create(ctx, bot)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(bot)
}
