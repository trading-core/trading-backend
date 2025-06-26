package bybit

type ObjectBase struct {
	ReturnCode      int                    `json:"retCode"`
	ReturnMessage   string                 `json:"retMsg"`
	ReturnExtraInfo map[string]interface{} `json:"retExtInfo"`
	UnixMilli       int64                  `json:"time"`
}

type ServerTime struct {
	ObjectBase
	ServerTimeResult ServerTimeResult `json:"result"`
}

type ServerTimeResult struct {
	UnixSecond string `json:"timeSecond"`
	UnixNano   string `json:"timeNano"`
}

type WalletBalance struct {
	ObjectBase
	WalletBalanceResult WalletBalanceResult `json:"result"`
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
