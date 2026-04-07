package chart

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

type RenderCombinedInput struct {
	Symbol      string
	TotalReturn float64
	Prices      []PricePoint
	Decisions   []DecisionMarker
	BollUpper   []IndicatorPoint
	BollMiddle  []IndicatorPoint
	BollLower   []IndicatorPoint
	SMA         []IndicatorPoint
	SMAPeriod   int
	RSI         []IndicatorPoint
	MACD        []IndicatorPoint
	MACDSignal  []IndicatorPoint
	RSIPeriod   int
	MACDFast    int
	MACDSlow    int
	MACDSignalN int
	Timezone    *time.Location
	Timeframe   string // e.g. "1h", "1d"; controls x-axis label format and separator granularity
}

func RenderCombined(input RenderCombinedInput, outputPath string) error {
	if len(input.Prices) == 0 {
		return errors.New("no price points to plot")
	}
	tz := input.Timezone
	if tz == nil {
		tz = time.UTC
	}

	const (
		width     = 1400
		height    = 980
		leftPad   = 82
		rightPad  = 30
		topPad    = 46
		bottomPad = 68
		panelGap  = 24
		priceH    = 420
		rsiH      = 180
		macdH     = 180
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
	priceTop := topPad
	priceBottom := priceTop + priceH
	rsiTop := priceBottom + panelGap
	rsiBottom := rsiTop + rsiH
	macdTop := rsiBottom + panelGap
	macdBottom := macdTop + macdH
	axisBottom := height - bottomPad

	daily := input.Timeframe == "1d" || input.Timeframe == "1w"

	var prices []PricePoint
	var decisions []DecisionMarker
	if daily {
		prices = input.Prices
		decisions = input.Decisions
	} else {
		prices = filterMarketHours(input.Prices, tz)
		if len(prices) == 0 {
			prices = input.Prices
		}
		decisions = filterDecisionMarketHours(input.Decisions, tz)
		if len(decisions) == 0 && len(input.Decisions) > 0 {
			decisions = input.Decisions
		}
	}

	tsToIndex := make(map[int64]int, len(prices))
	for i, p := range prices {
		tsToIndex[p.At.Unix()] = i
	}
	idxMin := 0
	idxMax := len(prices) - 1
	if idxMax <= 0 {
		idxMax = 1
	}
	xToPixel := func(idx int) int {
		fraction := float64(idx-idxMin) / float64(idxMax-idxMin)
		return plotLeft + int(fraction*float64(plotRight-plotLeft))
	}
	closestIndex := func(ts int64) int {
		if idx, ok := tsToIndex[ts]; ok {
			return idx
		}
		lo, hi := 0, len(prices)-1
		for lo < hi {
			mid := (lo + hi) / 2
			if prices[mid].At.Unix() < ts {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		return lo
	}

	var (
		axisColor    = color.RGBA{R: 120, G: 120, B: 120, A: 255}
		gridColor    = color.RGBA{R: 215, G: 215, B: 215, A: 255}
		labelColor   = color.RGBA{R: 70, G: 70, B: 70, A: 255}
		titleColor   = color.RGBA{R: 20, G: 20, B: 20, A: 255}
		priceColor   = color.RGBA{R: 35, G: 120, B: 230, A: 255}
		bollUp       = color.RGBA{R: 184, G: 90, B: 24, A: 255}
		bollMid      = color.RGBA{R: 95, G: 95, B: 95, A: 255}
		bollLow      = color.RGBA{R: 24, G: 144, B: 104, A: 255}
		buyColor     = color.RGBA{R: 25, G: 170, B: 70, A: 255}
		sellColor    = color.RGBA{R: 220, G: 40, B: 40, A: 255}
		sepColor     = color.RGBA{R: 180, G: 180, B: 180, A: 255}
		rsiColor     = color.RGBA{R: 160, G: 70, B: 30, A: 255}
		macdColor    = color.RGBA{R: 35, G: 120, B: 230, A: 255}
		signalColor  = color.RGBA{R: 220, G: 40, B: 40, A: 255}
		neutralColor = color.RGBA{R: 130, G: 130, B: 130, A: 255}
		smaColor     = color.RGBA{R: 130, G: 60, B: 200, A: 255}
	)

	firstDate := prices[0].At.In(tz).Format("2006-01-02")
	lastDate := prices[len(prices)-1].At.In(tz).Format("2006-01-02")
	dateStr := firstDate
	if lastDate != firstDate {
		dateStr = firstDate + " to " + lastDate
	}
	title := fmt.Sprintf("%s  |  %s  |  Return: %+.2f%%  |  RSI(%d) MACD(%d,%d,%d) SMA(%d)",
		input.Symbol, dateStr, input.TotalReturn*100, input.RSIPeriod, input.MACDFast, input.MACDSlow, input.MACDSignalN, input.SMAPeriod)
	drawText(img, title, plotLeft, 16, titleColor)

	// Price panel.
	yMin, yMax := prices[0].Close, prices[0].Close
	for _, p := range prices {
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
	priceY := func(v float64) int {
		fraction := (v - yMin) / (yMax - yMin)
		return priceBottom - int(fraction*float64(priceBottom-priceTop))
	}
	for _, tick := range niceTickValues(yMin, yMax, 6) {
		py := priceY(tick)
		drawLine(img, plotLeft, py, plotRight, py, gridColor)
		drawText(img, fmt.Sprintf("$%.2f", tick), plotLeft-62, py-5, labelColor)
	}
	drawLine(img, plotLeft, priceTop, plotLeft, priceBottom, axisColor)
	drawLine(img, plotLeft, priceBottom, plotRight, priceBottom, axisColor)
	drawText(img, "Price", plotLeft, priceTop-18, labelColor)
	for i := 1; i < len(prices); i++ {
		x1, y1 := xToPixel(i-1), priceY(prices[i-1].Close)
		x2, y2 := xToPixel(i), priceY(prices[i].Close)
		drawLine(img, x1, y1, x2, y2, priceColor)
	}
	for _, d := range decisions {
		x := xToPixel(closestIndex(d.At.Unix()))
		y := priceY(d.Price)
		if d.IsBuy {
			drawTriangle(img, x, y, 8, buyColor)
		} else {
			drawRing(img, x, y, 5, sellColor)
		}
	}
	drawIndicatorLine(img, input.BollUpper, closestIndex, xToPixel, priceY, bollUp)
	drawIndicatorLine(img, input.BollMiddle, closestIndex, xToPixel, priceY, bollMid)
	drawIndicatorLine(img, input.BollLower, closestIndex, xToPixel, priceY, bollLow)
	drawIndicatorLine(img, input.SMA, closestIndex, xToPixel, priceY, smaColor)

	// RSI panel.
	rsiY := func(v float64) int {
		fraction := v / 100.0
		return rsiBottom - int(fraction*float64(rsiBottom-rsiTop))
	}
	for _, tick := range []float64{30, 50, 70} {
		py := rsiY(tick)
		drawLine(img, plotLeft, py, plotRight, py, gridColor)
		drawText(img, fmt.Sprintf("%.0f", tick), plotLeft-30, py-5, labelColor)
	}
	drawLine(img, plotLeft, rsiTop, plotLeft, rsiBottom, axisColor)
	drawLine(img, plotLeft, rsiBottom, plotRight, rsiBottom, axisColor)
	drawText(img, "RSI", plotLeft, rsiTop-18, labelColor)
	drawIndicatorLine(img, input.RSI, closestIndex, xToPixel, rsiY, rsiColor)

	// MACD panel.
	macdMin, macdMax := rangeForSeries(input.MACD, input.MACDSignal)
	// Extend range to fit histogram bars (MACD − Signal can exceed either series alone).
	histMin, histMax := macdHistogramExtrema(input.MACD, input.MACDSignal)
	if histMin < macdMin {
		macdMin = histMin
	}
	if histMax > macdMax {
		macdMax = histMax
	}
	if macdMax == macdMin {
		macdMax = macdMin + 1
	}
	macdMargin := (macdMax - macdMin) * 0.15
	macdMin -= macdMargin
	macdMax += macdMargin
	macdY := func(v float64) int {
		fraction := (v - macdMin) / (macdMax - macdMin)
		return macdBottom - int(fraction*float64(macdBottom-macdTop))
	}
	for _, tick := range niceTickValues(macdMin, macdMax, 5) {
		py := macdY(tick)
		drawLine(img, plotLeft, py, plotRight, py, gridColor)
		drawText(img, fmt.Sprintf("%.3f", tick), plotLeft-54, py-5, labelColor)
	}
	zeroY := macdY(0)
	drawLine(img, plotLeft, zeroY, plotRight, zeroY, neutralColor)
	drawLine(img, plotLeft, macdTop, plotLeft, macdBottom, axisColor)
	drawLine(img, plotLeft, macdBottom, plotRight, macdBottom, axisColor)
	drawText(img, "MACD", plotLeft, macdTop-18, labelColor)
	drawMACDHistogram(img, input.MACD, input.MACDSignal, closestIndex, xToPixel, macdY)
	drawIndicatorLine(img, input.MACD, closestIndex, xToPixel, macdY, macdColor)
	drawIndicatorLine(img, input.MACDSignal, closestIndex, xToPixel, macdY, signalColor)

	// Shared x-axis: monthly separators for daily bars, daily separators for intraday.
	tickStep := len(prices) / 10
	if tickStep < 1 {
		tickStep = 1
	}
	prevPeriod := ""
	lastSepLabelRight := plotLeft - 1
	const sepLabelMinGap = 14
	for i, p := range prices {
		local := p.At.In(tz)
		var period, sepLabel string
		if daily {
			period = local.Format("2006-01")
			sepLabel = local.Format("Jan")
		} else {
			period = local.Format("2006-01-02")
			sepLabel = local.Format("01-02")
		}
		if period != prevPeriod && prevPeriod != "" {
			px := xToPixel(i)
			drawLine(img, px, priceTop, px, macdBottom, sepColor)
			labelX := px + 4
			labelRight := labelX + len(sepLabel)*7
			if labelX > lastSepLabelRight+sepLabelMinGap {
				drawText(img, sepLabel, labelX, priceTop+2, labelColor)
				lastSepLabelRight = labelRight
			}
		}
		prevPeriod = period

		if i%tickStep != 0 {
			continue
		}
		px := xToPixel(i)
		drawLine(img, px, macdBottom, px, axisBottom, gridColor)
		drawLine(img, px, axisBottom, px, axisBottom+5, axisColor)
		var label string
		if daily {
			label = local.Format("Jan 02")
		} else {
			label = local.Format("01-02 15:04")
		}
		drawText(img, label, px-len(label)*7/2, axisBottom+12, labelColor)
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
