package broker

type AccountType string

const (
	AccountTypeTastyTrade AccountType = "tastytrade"
)

type Account struct {
	Type AccountType `json:"account_type"`
	ID   string      `json:"account_id"`
}
