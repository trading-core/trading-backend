package indicator

import (
	"math"

	"github.com/kduong/trading-backend/internal/backtest/replay"
)

// ComputeADX computes the Average Directional Index using close-only data.
// Since only close prices are available in this pipeline, true range is
// approximated as |close[t] - close[t-1]|, and directional movement is
// derived from the sign of the close-to-close change.
//
// Algorithm:
//  1. For each bar t >= 1:
//     TR[t]  = |close[t] - close[t-1]|
//     +DM[t] = max(close[t]-close[t-1], 0)
//     -DM[t] = max(close[t-1]-close[t], 0)
//  2. Seed smoothed series with SMA of first period values, then apply
//     Wilder's smoothing: smooth = (prev*(period-1) + current) / period
//  3. +DI = 100 * smoothed(+DM) / smoothedTR
//     -DI = 100 * smoothed(-DM) / smoothedTR
//  4. DX  = 100 * |+DI - -DI| / (+DI + -DI)
//  5. ADX = Wilder's smoothing of DX over another period bars
//
// Minimum bars required: 2*period + 1.
// The first output point is anchored at prices[2*period].
func ComputeADX(prices []replay.PricePoint, period int) []Point {
	minBars := 2*period + 1
	if len(prices) < minBars || period < 2 {
		return nil
	}

	// Compute raw TR, +DM, -DM for bars 1..len-1.
	trRaw := make([]float64, len(prices)-1)
	plusDMRaw := make([]float64, len(prices)-1)
	minusDMRaw := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		diff := prices[i].Close - prices[i-1].Close
		trRaw[i-1] = math.Abs(diff)
		if diff > 0 {
			plusDMRaw[i-1] = diff
		}
		if diff < 0 {
			minusDMRaw[i-1] = -diff
		}
	}

	// Seed smoothed series with SMA of first period raw values.
	var trSum, plusSum, minusSum float64
	for i := 0; i < period; i++ {
		trSum += trRaw[i]
		plusSum += plusDMRaw[i]
		minusSum += minusDMRaw[i]
	}
	smoothedTR := trSum / float64(period)
	smoothedPlusDM := plusSum / float64(period)
	smoothedMinusDM := minusSum / float64(period)

	// Compute DX series using Wilder's smoothing from raw index period onward.
	// raw index i corresponds to prices index i+1.
	dxSeries := make([]float64, 0, len(trRaw)-period)
	for i := period; i < len(trRaw); i++ {
		smoothedTR = (smoothedTR*float64(period-1) + trRaw[i]) / float64(period)
		smoothedPlusDM = (smoothedPlusDM*float64(period-1) + plusDMRaw[i]) / float64(period)
		smoothedMinusDM = (smoothedMinusDM*float64(period-1) + minusDMRaw[i]) / float64(period)

		var dx float64
		if smoothedTR > 0 {
			plusDI := 100 * smoothedPlusDM / smoothedTR
			minusDI := 100 * smoothedMinusDM / smoothedTR
			diSum := plusDI + minusDI
			if diSum > 0 {
				dx = 100 * math.Abs(plusDI-minusDI) / diSum
			}
		}
		dxSeries = append(dxSeries, dx)
	}

	// Seed ADX with SMA of first period DX values.
	var dxSum float64
	for i := 0; i < period; i++ {
		dxSum += dxSeries[i]
	}
	adx := dxSum / float64(period)

	// Emit ADX from dxSeries index period onward, anchored at prices[2*period].
	output := make([]Point, 0, len(dxSeries)-period)
	output = append(output, Point{At: prices[2*period].At, Value: adx})
	for i := period; i < len(dxSeries); i++ {
		adx = (adx*float64(period-1) + dxSeries[i]) / float64(period)
		output = append(output, Point{At: prices[i+period+1].At, Value: adx})
	}

	return output
}
