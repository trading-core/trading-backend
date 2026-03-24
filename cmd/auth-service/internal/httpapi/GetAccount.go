package httpapi

import (
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/account"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) GetAccount(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	params := mux.Vars(request)
	accountID := params["account_id"]
	if len(accountID) == 0 {
		err = merry.New("account_id is required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("account_id is required")
		return
	}
	object, err := handler.accountStore.Get(ctx, accountID)
	if errors.Is(err, account.ErrAccountNotFound) {
		err = merry.New("account not found").WithHTTPCode(http.StatusNotFound).WithUserMessage("account not found")
		return
	}
	if err != nil {
		return
	}
	httputil.SendResponseJSON(responseWriter, http.StatusOK, object)
}
