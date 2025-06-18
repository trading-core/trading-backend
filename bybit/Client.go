package bybit

import "context"

type Client interface {
	GetServerTime(ctx context.Context) (output *ServerTime, err error)
	GetWalletBalance(ctx context.Context, input GetWalletBalanceInput) (output *WalletBalance, err error)
}

type ServerTime struct {
	ReturnCode       int                    `json:"retCode"`
	ReturnMessage    string                 `json:"retMsg"`
	ServerTimeResult ServerTimeResult       `json:"result"`
	ReturnExtraInfo  map[string]interface{} `json:"retExtInfo"`
	UnixMilli        int64                  `json:"time"`
}

type ServerTimeResult struct {
	UnixSecond string `json:"timeSecond"`
	UnixNano   string `json:"timeNano"`
}

type AccountType string

const (
	AccountTypeUnified  AccountType = "UNIFIED"
	AccountTypeContract AccountType = "CONTRACT"
	AccountTypeSpot     AccountType = "SPOT"
)

type GetWalletBalanceInput struct {
	AccountType        AccountType
	TimestampUnixMilli int64
	Coin               string
}

type WalletBalance struct {
	ReturnCode          int                    `json:"retCode"`
	ReturnMessage       string                 `json:"retMsg"`
	WalletBalanceResult WalletBalanceResult    `json:"result"`
	ReturnExtraInfo     map[string]interface{} `json:"retExtInfo"`
	Time                int64                  `json:"time"`
}

type WalletBalanceResult struct {
	List []WalletAccount `json:"list"`
}

type WalletAccount struct {
	TotalEquity            string     `json:"totalEquity"`
	AccountIMRate          string     `json:"accountIMRate"`
	TotalMarginBalance     string     `json:"totalMarginBalance"`
	TotalInitialMargin     string     `json:"totalInitialMargin"`
	AccountType            string     `json:"accountType"`
	TotalAvailableBalance  string     `json:"totalAvailableBalance"`
	AccountMMRate          string     `json:"accountMMRate"`
	TotalPerpUPL           string     `json:"totalPerpUPL"`
	TotalWalletBalance     string     `json:"totalWalletBalance"`
	AccountLTV             string     `json:"accountLTV"`
	TotalMaintenanceMargin string     `json:"totalMaintenanceMargin"`
	Coin                   []CoinInfo `json:"coin"`
}

type CoinInfo struct {
	AvailableToBorrow   string `json:"availableToBorrow"`
	Bonus               string `json:"bonus"`
	AccruedInterest     string `json:"accruedInterest"`
	AvailableToWithdraw string `json:"availableToWithdraw"`
	TotalOrderIM        string `json:"totalOrderIM"`
	Equity              string `json:"equity"`
	TotalPositionMM     string `json:"totalPositionMM"`
	USDValue            string `json:"usdValue"`
	SpotHedgingQty      string `json:"spotHedgingQty"`
	UnrealisedPnl       string `json:"unrealisedPnl"`
	CollateralSwitch    bool   `json:"collateralSwitch"`
	BorrowAmount        string `json:"borrowAmount"`
	TotalPositionIM     string `json:"totalPositionIM"`
	WalletBalance       string `json:"walletBalance"`
	CumRealisedPnl      string `json:"cumRealisedPnl"`
	Locked              string `json:"locked"`
	MarginCollateral    bool   `json:"marginCollateral"`
	Coin                string `json:"coin"`
}
