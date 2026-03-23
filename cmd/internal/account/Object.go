package account

type BrokerType string

const (
	BrokerTypeTastyTrade        BrokerType = "tastytrade"
	BrokerTypeTastyTradeSandbox BrokerType = "tastytrade_sandbox"
	BrokerTypeMockTest          BrokerType = "mocktest"
)

type Object struct {
	AccountID        string            `json:"account_id"`
	BrokerType       BrokerType        `json:"broker_type"`
	BrokerTastyTrade *BrokerTastyTrade `json:"broker_tasty_trade,omitempty"`
}

type BrokerTastyTrade struct {
	AccountNumber string `json:"account_number"`
}
