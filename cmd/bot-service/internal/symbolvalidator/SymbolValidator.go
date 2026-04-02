package symbolvalidator

import (
	"context"
	"errors"
)

var ErrSymbolNotTradableForBroker = errors.New("symbol is not tradable for broker")
var ErrUnsupportedBrokerForSymbolValidation = errors.New("unsupported broker type for symbol validation")

type SymbolValidator interface {
	Validate(ctx context.Context, brokerType string, symbol string) error
}

type NoopSymbolValidator struct{}

func (NoopSymbolValidator) Validate(ctx context.Context, brokerType string, symbol string) error {
	return nil
}
