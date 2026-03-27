package broker

type AccountType string

const (
	AccountTypeTastyTrade AccountType = "tastytrade"
)

type Account struct {
	Type       AccountType        `json:"type"`
	TastyTrade *AccountTastyTrade `json:"tastytrade,omitempty"`
}

type AccountTastyTrade struct {
	ID string `json:"id"`
}
