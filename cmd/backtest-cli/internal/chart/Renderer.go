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
	Timezone    *time.Location
}

func Render(input RenderInput, outputPath string) error {
	if len(input.Prices) == 0 {
		return errors.New("no price points to plot")
	}

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

	xMin := input.Prices[0].At.Unix()
	xMax := input.Prices[len(input.Prices)-1].At.Unix()
	if xMax == xMin {
		xMax = xMin + 1
	}

	yMin := input.Prices[0].Close
	yMax := input.Prices[0].Close
	for _, p := range input.Prices {
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

	xToPixel := func(ts int64) int {
		fraction := float64(ts-xMin) / float64(xMax-xMin)
		return plotLeft + int(fraction*float64(plotRight-plotLeft))
	}
	yToPixel := func(value float64) int {
		fraction := (value - yMin) / (yMax - yMin)
		return plotBottom - int(fraction*float64(plotBottom-plotTop))
	}

	var (
		axisColor  = color.RGBA{R: 120, G: 120, B: 120, A: 255}
		gridColor  = color.RGBA{R: 215, G: 215, B: 215, A: 255}
		labelColor = color.RGBA{R: 70, G: 70, B: 70, A: 255}
		titleColor = color.RGBA{R: 20, G: 20, B: 20, A: 255}
		priceColor = color.RGBA{R: 35, G: 120, B: 230, A: 255}
		buyColor   = color.RGBA{R: 25, G: 170, B: 70, A: 255}
		sellColor  = color.RGBA{R: 220, G: 40, B: 40, A: 255}
	)

	tz := input.Timezone
	if tz == nil {
		tz = time.UTC
	}

	date := input.Prices[0].At.In(tz).Format("2006-01-02")
	title := fmt.Sprintf("%s  |  %s  |  %s  |  Return: %+.2f%%",
		input.Symbol, input.Strategy, date, input.TotalReturn*100)
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

	for _, ts := range niceTimeTicksUnix(xMin, xMax, 9) {
		px := xToPixel(ts)
		if px < plotLeft || px > plotRight {
			continue
		}
		drawLine(img, px, plotTop, px, plotBottom, gridColor)
		drawLine(img, px, plotBottom, px, plotBottom+5, axisColor)
		label := time.Unix(ts, 0).In(tz).Format("15:04")
		drawText(img, label, px-len(label)*7/2, plotBottom+12, labelColor)
	}

	drawLine(img, plotLeft, plotBottom, plotRight, plotBottom, axisColor)
	drawLine(img, plotLeft, plotTop, plotLeft, plotBottom, axisColor)

	for i := 1; i < len(input.Prices); i++ {
		x1 := xToPixel(input.Prices[i-1].At.Unix())
		y1 := yToPixel(input.Prices[i-1].Close)
		x2 := xToPixel(input.Prices[i].At.Unix())
		y2 := yToPixel(input.Prices[i].Close)
		drawLine(img, x1, y1, x2, y2, priceColor)
		drawLine(img, x1, y1+1, x2, y2+1, priceColor)
	}

	for _, d := range input.Decisions {
		x := xToPixel(d.At.Unix())
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

func niceTimeTicksUnix(xMin, xMax int64, maxTicks int) []int64 {
	span := xMax - xMin
	if span <= 0 {
		return []int64{xMin}
	}
	step := int64(7200)
	for _, c := range []int64{60, 300, 600, 900, 1800, 3600, 7200} {
		if span/c <= int64(maxTicks) {
			step = c
			break
		}
	}
	start := (xMin/step + 1) * step
	var ticks []int64
	for ts := start; ts <= xMax; ts += step {
		ticks = append(ticks, ts)
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
