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

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type PricePoint struct {
	At    time.Time
	Close float64
}

type DecisionMarker struct {
	At    time.Time
	Price float64
	IsBuy bool
}

type RenderInput struct {
	Symbol      string
	Strategy    string
	TotalReturn float64
	Prices      []PricePoint
	Decisions   []DecisionMarker
	BollUpper   []IndicatorPoint
	BollMiddle  []IndicatorPoint
	BollLower   []IndicatorPoint
	Timezone    *time.Location
}

func Render(input RenderInput, outputPath string) error {
	if len(input.Prices) == 0 {
		return errors.New("no price points to plot")
	}

	tz := input.Timezone
	if tz == nil {
		tz = time.UTC
	}

	// Filter prices and decisions to market hours (09:30–16:00 ET) so
	// overnight gaps don't dominate the x-axis.
	prices := filterMarketHours(input.Prices, tz)
	if len(prices) == 0 {
		prices = input.Prices // fallback if nothing passes the filter
	}
	decisions := filterDecisionMarketHours(input.Decisions, tz)

	const (
		width    = 1400
		height   = 700
		leftPad  = 82
		rightPad = 30
		topPad   = 46
		botPad   = 68
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

	// Build a compressed x-axis that maps each data point to a sequential
	// index, eliminating overnight gaps between trading sessions.
	type indexedPrice struct {
		idx   int
		price PricePoint
	}
	indexed := make([]indexedPrice, len(prices))
	tsToIndex := make(map[int64]int, len(prices))
	for i, p := range prices {
		indexed[i] = indexedPrice{idx: i, price: p}
		tsToIndex[p.At.Unix()] = i
	}
	idxMin := 0
	idxMax := len(prices) - 1
	if idxMax <= 0 {
		idxMax = 1
	}

	yMin := prices[0].Close
	yMax := prices[0].Close
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
	priceMargin := (yMax - yMin) * 0.08
	yMin -= priceMargin
	yMax += priceMargin

	xToPixel := func(idx int) int {
		fraction := float64(idx-idxMin) / float64(idxMax-idxMin)
		return plotLeft + int(fraction*float64(plotRight-plotLeft))
	}
	yToPixel := func(value float64) int {
		fraction := (value - yMin) / (yMax - yMin)
		return plotBottom - int(fraction*float64(plotBottom-plotTop))
	}

	// Find the closest data index for a given timestamp.
	closestIndex := func(ts int64) int {
		if idx, ok := tsToIndex[ts]; ok {
			return idx
		}
		// Binary search for nearest point.
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
		axisColor  = color.RGBA{R: 120, G: 120, B: 120, A: 255}
		gridColor  = color.RGBA{R: 215, G: 215, B: 215, A: 255}
		labelColor = color.RGBA{R: 70, G: 70, B: 70, A: 255}
		titleColor = color.RGBA{R: 20, G: 20, B: 20, A: 255}
		priceColor = color.RGBA{R: 35, G: 120, B: 230, A: 255}
		bollUp     = color.RGBA{R: 184, G: 90, B: 24, A: 255}
		bollMid    = color.RGBA{R: 95, G: 95, B: 95, A: 255}
		bollLow    = color.RGBA{R: 24, G: 144, B: 104, A: 255}
		buyColor   = color.RGBA{R: 25, G: 170, B: 70, A: 255}
		sellColor  = color.RGBA{R: 220, G: 40, B: 40, A: 255}
		sepColor   = color.RGBA{R: 180, G: 180, B: 180, A: 255}
	)

	firstDate := prices[0].At.In(tz).Format("2006-01-02")
	lastDate := prices[len(prices)-1].At.In(tz).Format("2006-01-02")
	dateStr := firstDate
	if lastDate != firstDate {
		dateStr = firstDate + " to " + lastDate
	}
	title := fmt.Sprintf("%s  |  %s  |  %s  |  Return: %+.2f%%",
		input.Symbol, input.Strategy, dateStr, input.TotalReturn*100)
	drawText(img, title, plotLeft, 16, titleColor)

	for _, tick := range niceTickValues(yMin, yMax, 7) {
		py := yToPixel(tick)
		if py < plotTop || py > plotBottom {
			continue
		}
		drawLine(img, plotLeft, py, plotRight, py, gridColor)
		drawLine(img, plotLeft-5, py, plotLeft, py, axisColor)
		label := fmt.Sprintf("$%.2f", tick)
		drawText(img, label, plotLeft-len(label)*7-6, py-5, labelColor)
	}

	// Generate x-axis labels at regular index intervals, plus day separators.
	tickStep := len(prices) / 10
	if tickStep < 1 {
		tickStep = 1
	}
	prevDay := ""
	for i, p := range prices {
		day := p.At.In(tz).Format("2006-01-02")

		// Draw a vertical separator at each day boundary.
		if day != prevDay && prevDay != "" {
			px := xToPixel(i)
			drawLine(img, px, plotTop, px, plotBottom, sepColor)
			// Label the new date just below the top of the plot.
			drawText(img, day, px+4, plotTop+2, labelColor)
		}
		prevDay = day

		// Regular time tick labels along the bottom.
		if i%tickStep == 0 {
			px := xToPixel(i)
			if px >= plotLeft && px <= plotRight {
				drawLine(img, px, plotTop, px, plotBottom, gridColor)
				drawLine(img, px, plotBottom, px, plotBottom+5, axisColor)
				label := p.At.In(tz).Format("15:04")
				drawText(img, label, px-len(label)*7/2, plotBottom+12, labelColor)
			}
		}
	}

	drawLine(img, plotLeft, plotBottom, plotRight, plotBottom, axisColor)
	drawLine(img, plotLeft, plotTop, plotLeft, plotBottom, axisColor)

	for i := 1; i < len(prices); i++ {
		x1 := xToPixel(i - 1)
		y1 := yToPixel(prices[i-1].Close)
		x2 := xToPixel(i)
		y2 := yToPixel(prices[i].Close)
		drawLine(img, x1, y1, x2, y2, priceColor)
		drawLine(img, x1, y1+1, x2, y2+1, priceColor)
	}

	// Overlay Bollinger bands on the price panel.
	drawIndicatorLine(img, input.BollUpper, closestIndex, xToPixel, yToPixel, bollUp)
	drawIndicatorLine(img, input.BollMiddle, closestIndex, xToPixel, yToPixel, bollMid)
	drawIndicatorLine(img, input.BollLower, closestIndex, xToPixel, yToPixel, bollLow)

	for _, d := range decisions {
		idx := closestIndex(d.At.Unix())
		x := xToPixel(idx)
		y := yToPixel(d.Price)
		if d.IsBuy {
			drawTriangle(img, x, y, 8, buyColor)
		} else {
			drawRing(img, x, y, 5, sellColor)
		}
	}

	lx := plotRight - 100
	ly := plotTop + 10
	drawTriangle(img, lx+8, ly+8, 7, buyColor)
	drawText(img, "BUY", lx+20, ly+2, labelColor)
	drawRing(img, lx+8, ly+28, 5, sellColor)
	drawText(img, "SELL", lx+20, ly+22, labelColor)
	drawLine(img, lx, ly+44, lx+16, ly+44, bollUp)
	drawText(img, "B-U", lx+20, ly+38, labelColor)
	drawLine(img, lx, ly+58, lx+16, ly+58, bollMid)
	drawText(img, "B-M", lx+20, ly+52, labelColor)
	drawLine(img, lx, ly+72, lx+16, ly+72, bollLow)
	drawText(img, "B-L", lx+20, ly+66, labelColor)

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

func drawText(img *image.RGBA, text string, x, y int, c color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot: fixed.Point26_6{
			X: fixed.Int26_6(x * 64),
			Y: fixed.Int26_6((y + 10) * 64),
		},
	}
	d.DrawString(text)
}

func niceTickValues(dataMin, dataMax float64, maxTicks int) []float64 {
	span := dataMax - dataMin
	if span <= 0 {
		return []float64{dataMin}
	}
	rough := span / float64(maxTicks-1)
	mag := math.Pow(10, math.Floor(math.Log10(rough)))
	for _, f := range []float64{1, 2, 2.5, 5, 10} {
		if f*mag >= rough {
			mag = f * mag
			break
		}
	}
	start := math.Ceil(dataMin/mag) * mag
	var ticks []float64
	for v := start; v <= dataMax+mag*0.001; v += mag {
		ticks = append(ticks, math.Round(v/mag)*mag)
	}
	return ticks
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := intAbs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -intAbs(y1 - y0)
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

func intAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// isMarketHour returns true if t falls within 09:30–16:00 in the given timezone.
func isMarketHour(t time.Time, tz *time.Location) bool {
	local := t.In(tz)
	h, m, _ := local.Clock()
	mins := h*60 + m
	return mins >= 9*60+30 && mins <= 16*60
}

func filterMarketHours(prices []PricePoint, tz *time.Location) []PricePoint {
	out := make([]PricePoint, 0, len(prices))
	for _, p := range prices {
		if isMarketHour(p.At, tz) {
			out = append(out, p)
		}
	}
	return out
}

func filterDecisionMarketHours(decisions []DecisionMarker, tz *time.Location) []DecisionMarker {
	out := make([]DecisionMarker, 0, len(decisions))
	for _, d := range decisions {
		if isMarketHour(d.At, tz) {
			out = append(out, d)
		}
	}
	return out
}
