package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kduong/trading-backend/internal/broker/alpaca"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

type candle struct {
	At    time.Time
	Close float64
}

type decisionPoint struct {
	At       time.Time
	Price    float64
	Action   tradingstrategy.Action
	Quantity float64
	Reason   string
}

type result struct {
	Symbol        string
	Strategy      string
	StartingCash  float64
	EndingCash    float64
	EndingValue   float64
	TotalReturn   float64
	Prices        []candle
	Decisions     []decisionPoint
	FinalPosition float64
}

type sessionRange struct {
	date  string
	open  float64
	high  float64
	low   float64
	ready bool
}

func main() {
	var (
		alpacaLimit = flag.Int("alpaca-limit", 1000, "Alpaca bar limit")
		cash        = flag.Float64("cash", 10000, "Starting cash")
	)
	flag.Parse()

	symbol := strings.TrimSpace(config.EnvString("BACKTEST_SYMBOL", "AAPL"))
	strategyName := strings.TrimSpace(config.EnvString("BACKTEST_STRATEGY", "scalping"))
	alpacaTF := strings.TrimSpace(config.EnvString("BACKTEST_ALPACA_TIMEFRAME", "1Min"))
	alpacaStart := strings.TrimSpace(config.EnvString("BACKTEST_ALPACA_START", ""))
	alpacaEnd := strings.TrimSpace(config.EnvString("BACKTEST_ALPACA_END", ""))
	alpacaFeed := strings.TrimSpace(config.EnvString("BACKTEST_ALPACA_FEED", "iex"))
	outputPNG := strings.TrimSpace(config.EnvString("BACKTEST_OUTPUT", "backtest.png"))

	if *cash <= 0 {
		fatalf("-cash must be greater than zero")
	}
	if err := tradingstrategy.ValidateType(strategyName); err != nil {
		fatalf("invalid strategy from BACKTEST_STRATEGY: %v", err)
	}
	if *alpacaLimit <= 0 {
		fatalf("-alpaca-limit must be greater than zero")
	}

	candles, err := loadCandlesFromAlpaca(context.Background(), loadCandlesFromAlpacaInput{
		Symbol:    symbol,
		Timeframe: alpacaTF,
		Limit:     *alpacaLimit,
		Start:     alpacaStart,
		End:       alpacaEnd,
		Feed:      alpacaFeed,
	})
	if err != nil {
		fatalf("failed to load alpaca candles: %v", err)
	}
	if len(candles) == 0 {
		fatalf("alpaca returned no candle rows (symbol=%s timeframe=%s start=%q end=%q feed=%s limit=%d). This often means a non-trading window (weekend/holiday), too narrow time range, or unsupported feed for your account", symbol, alpacaTF, alpacaStart, alpacaEnd, alpacaFeed, *alpacaLimit)
	}

	res := runBacktest(symbol, strategyName, *cash, candles)
	if err := renderDecisionChart(res, outputPNG); err != nil {
		fatalf("failed to render chart: %v", err)
	}

	fmt.Printf("Backtest complete for %s (%s)\n", res.Symbol, res.Strategy)
	fmt.Printf("Rows: %d\n", len(res.Prices))
	fmt.Printf("Decisions: %d\n", len(res.Decisions))
	fmt.Printf("Starting cash: %.2f\n", res.StartingCash)
	fmt.Printf("Ending cash: %.2f\n", res.EndingCash)
	fmt.Printf("Ending value: %.2f\n", res.EndingValue)
	fmt.Printf("Total return: %.2f%%\n", res.TotalReturn*100)
	fmt.Printf("Output image: %s\n", outputPNG)
}

func runBacktest(symbol string, strategyName string, startingCash float64, candles []candle) result {
	strategy := tradingstrategy.New(strategyName)
	account := tradingstrategy.AccountSnapshot{
		CashBalance:      startingCash,
		BuyingPower:      startingCash,
		PositionQuantity: 0,
		HasOpenOrder:     false,
	}

	var (
		decisions []decisionPoint
		session   sessionRange
	)

	for _, c := range candles {
		snapshot := marketSnapshot(symbol, c, session)
		input := tradingstrategy.NewEvaluateInput(snapshot, account)
		decision := strategy.Evaluate(input)

		switch decision.Action {
		case tradingstrategy.ActionBuy:
			qty := decision.Quantity
			if qty <= 0 {
				break
			}
			cost := qty * c.Close
			if cost > account.CashBalance {
				qty = math.Floor(account.CashBalance / c.Close)
				cost = qty * c.Close
			}
			if qty > 0 {
				account.CashBalance -= cost
				account.BuyingPower = account.CashBalance
				account.PositionQuantity += qty
				decisions = append(decisions, decisionPoint{
					At:       c.At,
					Price:    c.Close,
					Action:   tradingstrategy.ActionBuy,
					Quantity: qty,
					Reason:   decision.Reason,
				})
			}
		case tradingstrategy.ActionSell:
			qty := decision.Quantity
			if qty <= 0 || qty > account.PositionQuantity {
				qty = account.PositionQuantity
			}
			if qty > 0 {
				proceeds := qty * c.Close
				account.CashBalance += proceeds
				account.BuyingPower = account.CashBalance
				account.PositionQuantity -= qty
				decisions = append(decisions, decisionPoint{
					At:       c.At,
					Price:    c.Close,
					Action:   tradingstrategy.ActionSell,
					Quantity: qty,
					Reason:   decision.Reason,
				})
			}
		}

		session = updateSession(session, c)
	}

	lastPrice := candles[len(candles)-1].Close
	endingValue := account.CashBalance + (account.PositionQuantity * lastPrice)
	return result{
		Symbol:        symbol,
		Strategy:      strategyName,
		StartingCash:  startingCash,
		EndingCash:    account.CashBalance,
		EndingValue:   endingValue,
		TotalReturn:   (endingValue - startingCash) / startingCash,
		Prices:        candles,
		Decisions:     decisions,
		FinalPosition: account.PositionQuantity,
	}
}

func marketSnapshot(symbol string, c candle, session sessionRange) tradingstrategy.MarketSnapshot {
	price := c.Close
	if !session.ready {
		return tradingstrategy.MarketSnapshot{
			Symbol:           symbol,
			LastTradePrice:   &price,
			SessionOpenPrice: 0,
			SessionHighPrice: 0,
			SessionLowPrice:  0,
			Now:              c.At,
		}
	}
	return tradingstrategy.MarketSnapshot{
		Symbol:           symbol,
		LastTradePrice:   &price,
		SessionOpenPrice: session.open,
		SessionHighPrice: session.high,
		SessionLowPrice:  session.low,
		Now:              c.At,
	}
}

func updateSession(session sessionRange, c candle) sessionRange {
	date := c.At.In(tradingstrategy.USMarketLocation).Format("2006-01-02")
	if session.date != date {
		return sessionRange{date: date, open: c.Close, high: c.Close, low: c.Close, ready: true}
	}
	if c.Close > session.high {
		session.high = c.Close
	}
	if c.Close < session.low {
		session.low = c.Close
	}
	return session
}

type loadCandlesFromAlpacaInput struct {
	Symbol    string
	Timeframe string
	Limit     int
	Start     string
	End       string
	Feed      string
}

func loadCandlesFromAlpaca(ctx context.Context, input loadCandlesFromAlpacaInput) ([]candle, error) {
	if input.Symbol == "" {
		return nil, errors.New("symbol is required for alpaca source")
	}
	if input.Timeframe == "" {
		return nil, errors.New("alpaca timeframe is required")
	}

	client := alpaca.ClientFromEnv()
	barsOutput, err := client.GetStockBars(ctx, alpaca.GetStockBarsInput{
		Symbol:    input.Symbol,
		Timeframe: input.Timeframe,
		Limit:     input.Limit,
		Feed:      input.Feed,
		Start:     input.Start,
		End:       input.End,
	})
	if err != nil {
		return nil, err
	}

	candles := make([]candle, 0, len(barsOutput.Bars))
	for _, bar := range barsOutput.Bars {
		at, err := parseTimestamp(bar.Time)
		if err != nil {
			return nil, fmt.Errorf("invalid alpaca bar time %q: %w", bar.Time, err)
		}
		candles = append(candles, candle{At: at, Close: bar.Close})
	}

	sort.Slice(candles, func(i, j int) bool {
		return candles[i].At.Before(candles[j].At)
	})
	return candles, nil
}

func parseTimestamp(value string) (time.Time, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, clean); err == nil {
			return ts, nil
		}
	}
	if unixSeconds, err := strconv.ParseInt(clean, 10, 64); err == nil {
		return time.Unix(unixSeconds, 0).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", clean)
}

func renderDecisionChart(res result, outputPath string) error {
	if len(res.Prices) == 0 {
		return errors.New("no price points to plot")
	}

	const (
		width    = 1400
		height   = 700
		leftPad  = 70
		rightPad = 20
		topPad   = 20
		botPad   = 60
	)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bg := color.RGBA{R: 248, G: 248, B: 248, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bg)
		}
	}

	plotLeft := leftPad
	plotRight := width - rightPad
	plotTop := topPad
	plotBottom := height - botPad

	xMin := res.Prices[0].At.Unix()
	xMax := res.Prices[len(res.Prices)-1].At.Unix()
	if xMax == xMin {
		xMax = xMin + 1
	}

	yMin := res.Prices[0].Close
	yMax := res.Prices[0].Close
	for _, p := range res.Prices {
		if p.Close < yMin {
			yMin = p.Close
		}
		if p.Close > yMax {
			yMax = p.Close
		}
	}
	if yMax == yMin {
		yMax = yMin + 1
	}
	margin := (yMax - yMin) * 0.08
	yMin -= margin
	yMax += margin

	xToPixel := func(ts int64) int {
		fraction := float64(ts-xMin) / float64(xMax-xMin)
		return plotLeft + int(fraction*float64(plotRight-plotLeft))
	}
	yToPixel := func(value float64) int {
		fraction := (value - yMin) / (yMax - yMin)
		return plotBottom - int(fraction*float64(plotBottom-plotTop))
	}

	axisColor := color.RGBA{R: 120, G: 120, B: 120, A: 255}
	line(img, plotLeft, plotBottom, plotRight, plotBottom, axisColor)
	line(img, plotLeft, plotTop, plotLeft, plotBottom, axisColor)

	priceColor := color.RGBA{R: 35, G: 120, B: 230, A: 255}
	for i := 1; i < len(res.Prices); i++ {
		x1 := xToPixel(res.Prices[i-1].At.Unix())
		y1 := yToPixel(res.Prices[i-1].Close)
		x2 := xToPixel(res.Prices[i].At.Unix())
		y2 := yToPixel(res.Prices[i].Close)
		line(img, x1, y1, x2, y2, priceColor)
	}

	buyColor := color.RGBA{R: 25, G: 170, B: 70, A: 255}
	sellColor := color.RGBA{R: 220, G: 40, B: 40, A: 255}
	for _, d := range res.Decisions {
		x := xToPixel(d.At.Unix())
		y := yToPixel(d.Price)
		if d.Action == tradingstrategy.ActionBuy {
			drawTriangle(img, x, y, 8, buyColor)
		}
		if d.Action == tradingstrategy.ActionSell {
			drawRing(img, x, y, 5, sellColor)
		}
	}

	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return err
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func line(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.Set(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawTriangle(img *image.RGBA, cx, cy, size int, c color.Color) {
	for y := 0; y <= size; y++ {
		half := (size - y) / 2
		for x := -half; x <= half; x++ {
			px := cx + x
			py := cy - y
			if image.Pt(px, py).In(img.Bounds()) {
				img.Set(px, py, c)
			}
		}
	}
}

func drawRing(img *image.RGBA, cx, cy, radius int, c color.Color) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			d := x*x + y*y
			if d >= (radius-1)*(radius-1) && d <= radius*radius {
				px := cx + x
				py := cy + y
				if image.Pt(px, py).In(img.Bounds()) {
					img.Set(px, py, c)
				}
			}
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
