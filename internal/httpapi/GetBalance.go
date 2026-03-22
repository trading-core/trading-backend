package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/kduong/trading-backend/internal/bybit"
)

type BalanceResponse struct {
	WalletBalance BalanceData `json:"walletBalance"`
}

type BalanceData struct {
	TotalWalletBalance    string `json:"totalWalletBalance"`
	TotalAvailableBalance string `json:"totalAvailableBalance"`
	TotalMarginBalance    string `json:"totalMarginBalance"`
}

func (httpAPI *HTTPAPI) GetBalance(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 10*time.Second)
	defer cancel()

	walletBalance, err := httpAPI.Client.GetWalletBalance(ctx, bybit.GetWalletBalanceInput{})
	if err != nil {
		http.Error(responseWriter, "Failed to fetch balance", http.StatusInternalServerError)
		return
	}

	if walletBalance == nil || len(walletBalance.WalletBalanceResult.List) == 0 {
		http.Error(responseWriter, "No wallet balance data", http.StatusInternalServerError)
		return
	}

	account := walletBalance.WalletBalanceResult.List[0]

	response := BalanceResponse{
		WalletBalance: BalanceData{
			TotalWalletBalance:    account.TotalWalletBalance,
			TotalAvailableBalance: account.TotalAvailableBalance,
			TotalMarginBalance:    account.TotalMarginBalance,
		},
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(response)
}
