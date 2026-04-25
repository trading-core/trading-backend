package pnlaggregator

import (
	"sort"

	"github.com/kduong/trading-backend/internal/broker"
)

type openLot struct {
	remainingQuantity float64
	unitValue         float64
}

// MatchRealizedPnL fills RealizedPnL on every closing trade by FIFO matching
// against earlier opening trades of the same symbol and side. Closes whose
// opens are not present in the dataset (e.g. the open happened before the
// requested window) fall back to using the close-leg cash for the unmatched
// portion so PnL does not silently drop to zero.
//
// Per-unit math uses signed cash values: a buy-to-open contributes a negative
// per-unit value and a sell-to-close a positive one, so realized = (close +
// open) * matched cleanly handles both long and short round-trips.
func MatchRealizedPnL(transactions []broker.Transaction) {
	sort.SliceStable(transactions, func(i, j int) bool {
		return transactions[i].ExecutedAt < transactions[j].ExecutedAt
	})
	longLotsBySymbol := map[string][]*openLot{}
	shortLotsBySymbol := map[string][]*openLot{}
	for index := range transactions {
		transaction := &transactions[index]
		if transaction.Type != "Trade" || transaction.Quantity == 0 {
			continue
		}
		unitValue := transaction.Value / transaction.Quantity
		switch transaction.Effect {
		case broker.OrderEffectOpen:
			lot := &openLot{remainingQuantity: transaction.Quantity, unitValue: unitValue}
			if transaction.Action == broker.OrderActionBuy {
				longLotsBySymbol[transaction.Symbol] = append(longLotsBySymbol[transaction.Symbol], lot)
			} else {
				shortLotsBySymbol[transaction.Symbol] = append(shortLotsBySymbol[transaction.Symbol], lot)
			}
		case broker.OrderEffectClose:
			queueBySymbol := longLotsBySymbol
			if transaction.Action == broker.OrderActionBuy {
				queueBySymbol = shortLotsBySymbol
			}
			remainingQuantity := transaction.Quantity
			realized := 0.0
			lots := queueBySymbol[transaction.Symbol]
			for remainingQuantity > 0 && len(lots) > 0 {
				front := lots[0]
				matched := front.remainingQuantity
				if remainingQuantity < matched {
					matched = remainingQuantity
				}
				realized += (unitValue + front.unitValue) * matched
				front.remainingQuantity -= matched
				remainingQuantity -= matched
				if front.remainingQuantity <= 0 {
					lots = lots[1:]
				}
			}
			queueBySymbol[transaction.Symbol] = lots
			if remainingQuantity > 0 {
				realized += unitValue * remainingQuantity
			}
			transaction.RealizedPnL = realized
		}
	}
}
