package account

type BrokerType string

const (
	BrokerTypeTastyTrade BrokerType = "tastytrade"
	BrokerTypeMockTest   BrokerType = "mocktest"
)

type Object struct {
	ID         string     `json:"id"`
	BrokerType BrokerType `json:"broker_type"`
}
