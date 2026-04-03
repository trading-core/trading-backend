package brokerfactory

import (
	"context"
	"errors"
	"strings"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/symbolvalidator"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type symbolValidationFunc func(ctx context.Context, symbol string) error

type SymbolValidator struct {
	validateByBrokerType map[broker.AccountType]symbolValidationFunc
}

type NewSymbolValidatorInput struct {
	TastyTradeClientFactory        tastytrade.ClientFactory
	TastyTradeSandboxClientFactory tastytrade.ClientFactory
}

func NewSymbolValidator(input NewSymbolValidatorInput) *SymbolValidator {
	return &SymbolValidator{
		validateByBrokerType: map[broker.AccountType]symbolValidationFunc{
			broker.AccountTypeTastyTrade: func(ctx context.Context, symbol string) error {
				return validateTastyTradeEquitySymbol(ctx, input.TastyTradeClientFactory.Create(), symbol)
			},
			broker.AccountTypeTastyTradeSandbox: func(ctx context.Context, symbol string) error {
				return validateTastyTradeEquitySymbol(ctx, input.TastyTradeSandboxClientFactory.Create(), symbol)
			},
		},
	}
}

func (validator *SymbolValidator) Validate(ctx context.Context, brokerType string, symbol string) error {
	validateSymbol, ok := validator.validateByBrokerType[broker.AccountType(brokerType)]
	if !ok {
		return symbolvalidator.ErrUnsupportedBrokerForSymbolValidation
	}
	return validateSymbol(ctx, symbol)
}

func validateTastyTradeEquitySymbol(ctx context.Context, client tastytrade.Client, symbol string) error {
	instrument, err := client.SearchSymbol(ctx, symbol)
	if err != nil {
		if errors.Is(err, tastytrade.ErrSymbolNotFound) {
			return symbolvalidator.ErrSymbolNotTradableForBroker
		}
		return err
	}
	if instrument == nil {
		return symbolvalidator.ErrSymbolNotTradableForBroker
	}
	if !strings.EqualFold(instrument.Symbol, symbol) {
		return symbolvalidator.ErrSymbolNotTradableForBroker
	}
	if instrument.InstrumentType != "Equity" {
		return symbolvalidator.ErrSymbolNotTradableForBroker
	}
	return nil
}
