package account

import (
	"context"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
	"github.com/kduong/trading-backend/internal/logger"
)

type ValidationDecorator struct {
	decorated        Store
	tastyTradeClient tastytrade.Client
}

func (decorator *ValidationDecorator) Create(ctx context.Context, input CreateInput) error {
	return decorator.decorated.Create(ctx, input)
}

func (decorator *ValidationDecorator) LinkBrokerAccount(ctx context.Context, input LinkBrokerAccountInput) error {
	validate, ok := validateBrokerAccountByType[input.BrokerAccount.Type]
	if !ok {
		logger.Fatalf("Unsupported broker type: %s", input.BrokerAccount.Type)
		return nil
	}
	if err := validate(decorator, ctx, input.BrokerAccount); err != nil {
		return err
	}
	return decorator.decorated.LinkBrokerAccount(ctx, input)
}

func (decorator *ValidationDecorator) Get(ctx context.Context, input GetInput) (*Account, error) {
	return decorator.decorated.Get(ctx, input)
}

func (decorator *ValidationDecorator) List(ctx context.Context) ([]*Account, error) {
	return decorator.decorated.List(ctx)
}

type validateBrokerAccountFunc func(decorator *ValidationDecorator, ctx context.Context, brokerAccount *broker.Account) error

var validateBrokerAccountByType = map[broker.AccountType]validateBrokerAccountFunc{
	broker.AccountTypeTastyTrade: (*ValidationDecorator).validateTastyTradeAccount,
}

func (decorator *ValidationDecorator) validateTastyTradeAccount(ctx context.Context, brokerAccount *broker.Account) error {
	_, err := decorator.tastyTradeClient.GetAccountBalance(ctx, brokerAccount.TastyTrade.ID)
	return err
}
