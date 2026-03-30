package accountservice

import (
	"context"
	"errors"
	"time"

	"github.com/kduong/trading-backend/internal/config"
)

var (
	ErrAccountNotFound  = errors.New("account not found")
	ErrAccountForbidden = errors.New("account forbidden")
	ErrServerError      = errors.New("server error")
)

type Client interface {
	GetAccount(ctx context.Context, accountID string) (*Account, error)
	GetAccountBalance(ctx context.Context, accountID string) (*Balance, error)
}

type Account struct {
	ID           string         `json:"account_id"`
	UserID       string         `json:"user_id"`
	Name         string         `json:"name"`
	BrokerLinked bool           `json:"broker_linked"`
	Broker       *BrokerAccount `json:"broker_account,omitempty"`
}

type BrokerAccount struct {
	Type string `json:"account_type"`
	ID   string `json:"account_id"`
}

type Balance struct {
	NetLiquidatingValue float64 `json:"net_liquidating_value"`
	CashBalance         float64 `json:"cash_balance"`
	EquityBuyingPower   float64 `json:"equity_buying_power"`
	Currency            string  `json:"currency"`
}

func ClientFromEnv() Client {
	implementation := config.EnvStringOrFatal("ACCOUNT_SERVICE_CLIENT_IMPLEMENTATION")
	switch implementation {
	case "HTTP":
		return NewHTTPClient(NewHTTPClientInput{
			Timeout: config.EnvDuration("ACCOUNT_SERVICE_HTTP_CLIENT_TIMEOUT", 20*time.Second),
			BaseURL: config.EnvURLOrFatal("ACCOUNT_SERVICE"),
		})
	default:
		panic("invalid account service client implementation: " + implementation)
	}
}
