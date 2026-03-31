package broker

type AccountType string

const (
	AccountTypeTastyTrade        AccountType = "tastytrade"
	AccountTypeTastyTradeSandbox AccountType = "tastytrade_sandbox"
)

type Account struct {
	Type AccountType `json:"account_type"`
	ID   string      `json:"account_id"`
}
