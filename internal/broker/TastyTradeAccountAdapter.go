package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type TastyTradeAccountAdapter struct {
	accountID string
	client    tastytrade.Client
}

type NewTastyTradeAccountAdapterInput struct {
	AccountID string
	Client    tastytrade.Client
}

func NewTastyTradeAccountAdapter(input NewTastyTradeAccountAdapterInput) *TastyTradeAccountAdapter {
	return &TastyTradeAccountAdapter{
		accountID: input.AccountID,
		client:    input.Client,
	}
}

func (adapter *TastyTradeAccountAdapter) GetBalance(ctx context.Context) (output *GetBalanceOutput, err error) {
	tastyTradeAccountBalance, err := adapter.client.GetAccountBalance(ctx, adapter.accountID)
	if err != nil {
		return
	}
	data := tastyTradeAccountBalance.Data
	netLiquidatingValue, err := strconv.ParseFloat(data.NetLiquidatingValue, 64)
	if err != nil {
		return
	}
	cashBalance, err := strconv.ParseFloat(data.CashBalance, 64)
	if err != nil {
		return
	}
	equityBuyingPower, err := strconv.ParseFloat(data.EquityBuyingPower, 64)
	if err != nil {
		return
	}
	output = &GetBalanceOutput{
		NetLiquidatingValue: netLiquidatingValue,
		CashBalance:         cashBalance,
		EquityBuyingPower:   equityBuyingPower,
		Currency:            data.Currency,
	}
	return
}

func (adapter *TastyTradeAccountAdapter) GetEquityPosition(ctx context.Context, symbol string) (output *GetEquityPositionOutput, err error) {
	positions, err := adapter.client.GetAccountPositions(ctx, adapter.accountID)
	if err != nil {
		return
	}
	output = &GetEquityPositionOutput{}
	for _, pos := range positions.Data.Items {
		if pos.Symbol != symbol || pos.InstrumentType != "Equity" {
			continue
		}
		qty, parseErr := parsePositionQuantity(pos.Quantity)
		if parseErr != nil {
			continue
		}
		if pos.QuantityDirection == "Long" {
			output.Quantity += qty
		} else if pos.QuantityDirection == "Short" {
			output.Quantity -= qty
		}
	}
	return
}

func parsePositionQuantity(value any) (float64, error) {
	switch typed := value.(type) {
	case string:
		return strconv.ParseFloat(typed, 64)
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	default:
		return 0, strconv.ErrSyntax
	}
}

func (adapter *TastyTradeAccountAdapter) PlaceOrder(ctx context.Context, input PlaceOrderInput) (output *PlaceOrderOutput, err error) {
	var action string
	switch input.Action {
	case OrderActionBuy:
		action = "Buy to Open"
	case OrderActionSell:
		action = "Sell to Close"
	default:
		err = fmt.Errorf("unknown order action: %s", input.Action)
		return
	}
	result, err := adapter.client.PlaceEquityOrder(ctx, tastytrade.PlaceEquityOrderInput{
		AccountID: adapter.accountID,
		Symbol:    input.Symbol,
		Action:    action,
		Quantity:  input.Quantity,
	})
	if err != nil {
		return
	}
	output = &PlaceOrderOutput{
		OrderID: result.OrderID,
	}
	return
}

func (adapter *TastyTradeAccountAdapter) HasPendingOrder(ctx context.Context, symbol string) (bool, error) {
	liveOrders, err := adapter.client.GetLiveOrders(ctx, adapter.accountID)
	if err != nil {
		return false, err
	}
	for _, order := range liveOrders.Data.Items {
		if order.IsTerminal() {
			continue
		}
		for _, leg := range order.Legs {
			if leg.Symbol == symbol {
				return true, nil
			}
		}
	}
	return false, nil
}
