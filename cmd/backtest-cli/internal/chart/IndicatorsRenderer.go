package chart

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"
)

type IndicatorPoint struct {
	At    time.Time
	Value float64
}

type RenderIndicatorsInput struct {
	Symbol      string
	Timeline    []time.Time
	RSI         []IndicatorPoint
	MACD        []IndicatorPoint
	MACDSignal  []IndicatorPoint
	RSIPeriod   int
	MACDFast    int
	MACDSlow    int
	MACDSignalN int
	Timezone    *time.Location
	Timeframe   string // e.g. "1h", "1d"; controls x-axis label format
}

func RenderIndicators(input RenderIndicatorsInput, outputPath string) error {
	if len(input.Timeline) == 0 {
		return errors.New("no timeline points to plot indicators")
	}

	tz := input.Timezone
	if tz == nil {
		tz = time.UTC
	}

	const (
		width      = 1400
		height     = 760
		leftPad    = 82
		rightPad   = 30
		topPad     = 46
		bottomPad  = 68
		panelGap   = 24
		rsiHeight  = 260
		macdHeight = 260
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
	rsiTop := topPad
	rsiBottom := rsiTop + rsiHeight
	macdTop := rsiBottom + panelGap
	macdBottom := macdTop + macdHeight
	axisBottom := height - bottomPad

	tsToIndex := make(map[int64]int, len(input.Timeline))
	for i, ts := range input.Timeline {
		tsToIndex[ts.Unix()] = i
	}
	idxMin := 0
	idxMax := len(input.Timeline) - 1
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
		lo, hi := 0, len(input.Timeline)-1
		for lo < hi {
			mid := (lo + hi) / 2
			if input.Timeline[mid].Unix() < ts {
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
		rsiColor     = color.RGBA{R: 160, G: 70, B: 30, A: 255}
		macdColor    = color.RGBA{R: 35, G: 120, B: 230, A: 255}
		signalColor  = color.RGBA{R: 220, G: 40, B: 40, A: 255}
		neutralColor = color.RGBA{R: 130, G: 130, B: 130, A: 255}
	)

	firstDate := input.Timeline[0].In(tz).Format("2006-01-02")
	lastDate := input.Timeline[len(input.Timeline)-1].In(tz).Format("2006-01-02")
	dateStr := firstDate
	if lastDate != firstDate {
		dateStr = firstDate + " to " + lastDate
	}
	title := fmt.Sprintf("%s  |  %s  |  RSI(%d) MACD(%d,%d,%d)",
		input.Symbol, dateStr, input.RSIPeriod, input.MACDFast, input.MACDSlow, input.MACDSignalN)
	drawText(img, title, plotLeft, 16, titleColor)

	// RSI panel.
	rsiMin := 0.0
	rsiMax := 100.0
	rsiY := func(v float64) int {
		fraction := (v - rsiMin) / (rsiMax - rsiMin)
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
	if macdMin == macdMax {
		macdMax = macdMin + 1
	}
	margin := (macdMax - macdMin) * 0.15
	macdMin -= margin
	macdMax += margin
	macdY := func(v float64) int {
		fraction := (v - macdMin) / (macdMax - macdMin)
		return macdBottom - int(fraction*float64(macdBottom-macdTop))
	}
	for _, tick := range niceTickValues(macdMin, macdMax, 6) {
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

	daily := input.Timeframe == "1d" || input.Timeframe == "1w"

	// Shared x-axis labels at the bottom.
	tickStep := len(input.Timeline) / 10
	if tickStep < 1 {
		tickStep = 1
	}
	for i, ts := range input.Timeline {
		if i%tickStep != 0 {
			continue
		}
		px := xToPixel(i)
		drawLine(img, px, macdBottom, px, axisBottom, gridColor)
		drawLine(img, px, axisBottom, px, axisBottom+5, axisColor)
		var label string
		if daily {
			label = ts.In(tz).Format("Jan 02")
		} else {
			label = ts.In(tz).Format("01-02 15:04")
		}
		drawText(img, label, px-len(label)*7/2, axisBottom+12, labelColor)
	}
	// Legend.
	lx := plotRight - 140
	ly := topPad + 8
	drawLine(img, lx, ly+8, lx+18, ly+8, rsiColor)
	drawText(img, "RSI", lx+24, ly+2, labelColor)
	drawLine(img, lx, ly+28, lx+18, ly+28, macdColor)
	drawText(img, "MACD", lx+24, ly+22, labelColor)
	drawLine(img, lx, ly+48, lx+18, ly+48, signalColor)
	drawText(img, "Signal", lx+24, ly+42, labelColor)

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

// drawMACDHistogram draws histogram bars (MACD − Signal) behind the MACD lines.
// Bars are colored by direction and momentum, matching TradingView convention:
//   - rising positive  → strong bull (teal)
//   - falling positive → weak bull   (light teal)
//   - falling negative → strong bear (red)
//   - rising negative  → weak bear   (light red)
func drawMACDHistogram(img *image.RGBA, macd []IndicatorPoint, signal []IndicatorPoint, closestIndex func(int64) int, xToPixel func(int) int, yToPixel func(float64) int) {
	if len(macd) == 0 || len(signal) == 0 {
		return
	}
	sigByTs := make(map[int64]float64, len(signal))
	for _, p := range signal {
		sigByTs[p.At.Unix()] = p.Value
	}
	type bar struct {
		x     int
		value float64
	}
	bars := make([]bar, 0, len(macd))
	for _, p := range macd {
		sig, ok := sigByTs[p.At.Unix()]
		if !ok {
			continue
		}
		bars = append(bars, bar{x: xToPixel(closestIndex(p.At.Unix())), value: p.Value - sig})
	}
	if len(bars) == 0 {
		return
	}
	barHalf := 1
	if len(bars) > 1 {
		span := bars[len(bars)-1].x - bars[0].x
		if span > 0 {
			barHalf = span / len(bars) / 3
		}
		if barHalf < 1 {
			barHalf = 1
		}
		if barHalf > 6 {
			barHalf = 6
		}
	}
	zeroY := yToPixel(0)
	var (
		strongBull = color.RGBA{R: 34, G: 180, B: 34, A: 255}   // green
		weakBull   = color.RGBA{R: 144, G: 220, B: 144, A: 255} // light green
		strongBear = color.RGBA{R: 220, G: 40, B: 40, A: 255}   // red
		weakBear   = color.RGBA{R: 240, G: 160, B: 160, A: 255} // light red
	)
	for i, b := range bars {
		prevVal := b.value
		if i > 0 {
			prevVal = bars[i-1].value
		}
		var c color.RGBA
		switch {
		case b.value >= 0 && b.value >= prevVal:
			c = strongBull
		case b.value >= 0:
			c = weakBull
		case b.value < 0 && b.value <= prevVal:
			c = strongBear
		default:
			c = weakBear
		}
		top, bot := yToPixel(b.value), zeroY
		if top > bot {
			top, bot = bot, top
		}
		for y := top; y <= bot; y++ {
			for dx := -barHalf; dx <= barHalf; dx++ {
				if pt := image.Pt(b.x+dx, y); pt.In(img.Bounds()) {
					img.Set(b.x+dx, y, c)
				}
			}
		}
	}
}

// macdHistogramExtrema returns the min and max of (MACD − Signal) across matched timestamps.
// Used to extend the y-axis range so histogram bars don't get clipped.
func macdHistogramExtrema(macd []IndicatorPoint, signal []IndicatorPoint) (float64, float64) {
	sigByTs := make(map[int64]float64, len(signal))
	for _, p := range signal {
		sigByTs[p.At.Unix()] = p.Value
	}
	minV, maxV := 0.0, 0.0
	first := true
	for _, p := range macd {
		sig, ok := sigByTs[p.At.Unix()]
		if !ok {
			continue
		}
		h := p.Value - sig
		if first {
			minV, maxV = h, h
			first = false
		} else {
			if h < minV {
				minV = h
			}
			if h > maxV {
				maxV = h
			}
		}
	}
	return minV, maxV
}

func drawIndicatorLine(img *image.RGBA, points []IndicatorPoint, closestIndex func(ts int64) int, xToPixel func(int) int, yToPixel func(float64) int, c color.Color) {
	if len(points) < 2 {
		return
	}
	for i := 1; i < len(points); i++ {
		x1 := xToPixel(closestIndex(points[i-1].At.Unix()))
		y1 := yToPixel(points[i-1].Value)
		x2 := xToPixel(closestIndex(points[i].At.Unix()))
		y2 := yToPixel(points[i].Value)
		drawLine(img, x1, y1, x2, y2, c)
	}
}

func rangeForSeries(a []IndicatorPoint, b []IndicatorPoint) (float64, float64) {
	if len(a) == 0 && len(b) == 0 {
		return -1, 1
	}
	minV := math.MaxFloat64
	maxV := -math.MaxFloat64
	for _, p := range a {
		if p.Value < minV {
			minV = p.Value
		}
		if p.Value > maxV {
			maxV = p.Value
		}
	}
	for _, p := range b {
		if p.Value < minV {
			minV = p.Value
		}
		if p.Value > maxV {
			maxV = p.Value
		}
	}
	if minV == math.MaxFloat64 {
		return -1, 1
	}
	return minV, maxV
}
