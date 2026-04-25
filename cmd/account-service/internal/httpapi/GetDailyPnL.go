package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/cmd/account-service/internal/pnlaggregator"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httpx"
)

const dailyPnLDateLayout = "2006-01-02"
const dailyPnLMaxRangeDays = 366

// dailyPnLMatchingLookbackDays widens the broker fetch backwards from the
// requested `from` date so that closes within the requested window can be
// FIFO-matched against opens that happened earlier. The extra rows are
// discarded after matching and never appear in the response.
const dailyPnLMatchingLookbackDays = 365

type GetDailyPnLResponse struct {
	Currency string                 `json:"currency"`
	Days     []pnlaggregator.DailyPnL `json:"days"`
}

func (handler *Handler) GetDailyPnL(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	accountID := vars["account_id"]

	from := request.URL.Query().Get("from")
	to := request.URL.Query().Get("to")
	if from == "" || to == "" {
		err = merry.New("from and to are required (YYYY-MM-DD)").WithHTTPCode(http.StatusBadRequest)
		return
	}
	fromDate, parseErr := time.Parse(dailyPnLDateLayout, from)
	if parseErr != nil {
		err = merry.New("from must be YYYY-MM-DD").WithHTTPCode(http.StatusBadRequest)
		return
	}
	toDate, parseErr := time.Parse(dailyPnLDateLayout, to)
	if parseErr != nil {
		err = merry.New("to must be YYYY-MM-DD").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if toDate.Before(fromDate) {
		err = merry.New("to must be on or after from").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if toDate.Sub(fromDate).Hours()/24 > dailyPnLMaxRangeDays {
		err = merry.New("date range must be at most 366 days").WithHTTPCode(http.StatusBadRequest)
		return
	}

	account, err := handler.accountStoreQueryHandler.Get(ctx, accountstore.GetInput{
		AccountID: accountID,
	})
	if err != nil {
		err = merryErrorByAccountStoreError[err]
		return
	}
	if !account.BrokerLinked {
		err = merry.New("account is not linked to a broker").WithHTTPCode(http.StatusBadRequest)
		return
	}

	accountClient := handler.brokerAccountClientFactory.Get(ctx, account.BrokerAccount)
	matchingFrom := fromDate.AddDate(0, 0, -dailyPnLMatchingLookbackDays).Format(dailyPnLDateLayout)
	transactionsOutput, err := accountClient.GetTransactions(ctx, broker.GetTransactionsInput{
		From: matchingFrom,
		To:   to,
	})
	if err != nil {
		return
	}
	pnlaggregator.MatchRealizedPnL(transactionsOutput.Transactions)
	withinRequestedWindow := pnlaggregator.FilterByDateRange(transactionsOutput.Transactions, from, to)
	aggregated := pnlaggregator.Aggregate(withinRequestedWindow)

	balance, err := accountClient.GetBalance(ctx)
	currency := "USD"
	if err == nil && balance != nil && balance.Currency != "" {
		currency = balance.Currency
	}
	err = nil

	response := GetDailyPnLResponse{
		Currency: currency,
		Days:     aggregated.Days,
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(response)
	fatal.OnErrorUnlessDone(ctx, err)
}
