package bybit

type AccountType string

const (
	AccountTypeUnified  AccountType = "UNIFIED"
	AccountTypeContract AccountType = "CONTRACT"
	AccountTypeSpot     AccountType = "SPOT"
)

type Currency string

const (
	CurrencyBitcoin  Currency = "BTC"
	CurrencyEthereum Currency = "ETH"
	CurrencyUSDC     Currency = "USDC"
)
