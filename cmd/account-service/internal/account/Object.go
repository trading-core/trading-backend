package account

type BrokerType string

const (
	BrokerTypeTastyTrade BrokerType = "tastytrade"
)

type Object struct {
	AccountID       string     `json:"account_id"`
	BrokerType      BrokerType `json:"broker_type"`
	BrokerAccountID string     `json:"broker_account_id"`
}
