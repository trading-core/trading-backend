package sweeper

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/backtest"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/backtestconfig"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

type Sweeper struct {
	TakeProfitValues   []float64
	PositionValues     []float64
	SessionStartValues []int
	SessionEndValues   []int
}

func (sweeper *Sweeper) Run(cfg backtestconfig.Config, prices []replay.PricePoint, events []replay.Event, outputDir string) {
	// Split data into per-day segments so each param combo is tested across
	// multiple sessions to avoid single-day overfitting.
	days := splitByTradingDay(events, prices)
	fatal.Unlessf(len(days) > 0, "no trading days found in data")
	fmt.Fprintf(os.Stderr, "sweep: %d trading day(s) detected\n", len(days))

	// Window sizes for multi-day sessions. Window=1 is single-day (original
	// behaviour). Larger windows carry position/cash across consecutive days.
	var windowSizes []int
	for w := 1; w <= len(days); w++ {
		windowSizes = append(windowSizes, w)
	}

	type sweepResult struct {
		TakeProfit   float64
		Position     float64
		Start        int
		End          int
		WindowDays   int
		AvgReturn    float64
		WinWindows   int
		TotalWindows int
		TotalTrades  int
	}

	var results []sweepResult

	combos := len(sweeper.TakeProfitValues) * len(sweeper.PositionValues) * len(sweeper.SessionStartValues) * len(sweeper.SessionEndValues) * len(windowSizes)
	run := 0
	for _, tp := range sweeper.TakeProfitValues {
		for _, pos := range sweeper.PositionValues {
			for _, ss := range sweeper.SessionStartValues {
				for _, se := range sweeper.SessionEndValues {
					for _, ws := range windowSizes {
						run++
						sweepCfg := cfg
						sweepCfg.TradingParameters.TakeProfitPct = tp
						sweepCfg.TradingParameters.MaxPositionFraction = pos
						sweepCfg.TradingParameters.SessionStart = ss
						sweepCfg.TradingParameters.SessionEnd = se
						var totalReturn float64
						var winWindows, totalTrades, windows int
						for i := 0; i+ws <= len(days); i++ {
							windowEvents, windowPrices := mergeWindow(days[i : i+ws])
							res := backtest.Run(sweepCfg, windowPrices, windowEvents)
							totalReturn += res.TotalReturn
							totalTrades += len(res.Decisions)
							if res.TotalReturn > 0 {
								winWindows++
							}
							windows++
						}
						if windows > 0 {
							results = append(results, sweepResult{
								TakeProfit:   tp,
								Position:     pos,
								Start:        ss,
								End:          se,
								WindowDays:   ws,
								AvgReturn:    totalReturn / float64(windows),
								WinWindows:   winWindows,
								TotalWindows: windows,
								TotalTrades:  totalTrades,
							})
						}
						if run%50 == 0 || run == combos {
							fmt.Fprintf(os.Stderr, "sweep: %d/%d combos\n", run, combos)
						}
					}
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].AvgReturn > results[j].AvgReturn
	})

	csvPath := fmt.Sprintf("%s/sweep-results.csv", outputDir)
	f, err := os.Create(csvPath)
	if err != nil {
		fatal.OnError(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"TakeProfit%", "Position%", "SessionStartHour(ET)", "SessionEndHour(ET)", "WindowDays", "AvgReturn%", "WinRate%", "Windows", "Trades"})
	for _, r := range results {
		winRate := float64(r.WinWindows) / float64(r.TotalWindows) * 100
		w.Write([]string{
			fmt.Sprintf("%.2f", r.TakeProfit*100),
			fmt.Sprintf("%.1f", r.Position*100),
			strconv.Itoa(r.Start),
			strconv.Itoa(r.End),
			strconv.Itoa(r.WindowDays),
			fmt.Sprintf("%.4f", r.AvgReturn*100),
			fmt.Sprintf("%.1f", winRate),
			strconv.Itoa(r.TotalWindows),
			strconv.Itoa(r.TotalTrades),
		})
	}
	w.Flush()
	fatal.OnError(w.Error())

	fmt.Fprintf(os.Stderr, "sweep results written to %s (%d rows)\n", csvPath, len(results))
}

// mergeWindow concatenates events and prices from consecutive trading days into
// a single contiguous slice for multi-day backtest sessions.
func mergeWindow(days []tradingDay) ([]replay.Event, []replay.PricePoint) {
	var events []replay.Event
	var prices []replay.PricePoint
	for _, d := range days {
		events = append(events, d.events...)
		prices = append(prices, d.prices...)
	}
	return events, prices
}

// tradingDay holds one day's worth of events and the corresponding price series.
type tradingDay struct {
	date   string // "2006-01-02"
	events []replay.Event
	prices []replay.PricePoint
}

// splitByTradingDay partitions events and prices into per-calendar-day segments
// using the US Eastern timezone so that a single market session stays together.
func splitByTradingDay(events []replay.Event, prices []replay.PricePoint) []tradingDay {
	dayEvents := make(map[string][]replay.Event)
	for _, e := range events {
		d := e.At.In(tradingstrategy.USMarketLocation).Format("2006-01-02")
		dayEvents[d] = append(dayEvents[d], e)
	}
	dayPrices := make(map[string][]replay.PricePoint)
	for _, p := range prices {
		d := p.At.In(tradingstrategy.USMarketLocation).Format("2006-01-02")
		dayPrices[d] = append(dayPrices[d], p)
	}

	var dates []string
	for d := range dayEvents {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	var days []tradingDay
	for _, d := range dates {
		p := dayPrices[d]
		if len(p) == 0 {
			p = replay.CandlesFromEvents(dayEvents[d])
		}
		if len(p) == 0 {
			continue
		}
		days = append(days, tradingDay{date: d, events: dayEvents[d], prices: p})
	}
	return days
}
