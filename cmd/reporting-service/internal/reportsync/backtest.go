package reportsync

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kduong/trading-backend/cmd/reporting-service/internal/jobstore"
	"github.com/kduong/trading-backend/cmd/storage-service/pkg/storageservice"
	"github.com/kduong/trading-backend/internal/backtest/backtest"
	"github.com/kduong/trading-backend/internal/backtest/backtestconfig"
	"github.com/kduong/trading-backend/internal/backtest/chart"
	"github.com/kduong/trading-backend/internal/backtest/indicator"
	"github.com/kduong/trading-backend/internal/backtest/replay"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

const JobKind = "backtest"

// BacktestParameters are the user-supplied inputs for a backtest job,
// stored as the job's parameters map.
type BacktestParameters struct {
	Symbol     string                         `json:"symbol"`
	Start      string                         `json:"start"`
	End        string                         `json:"end"`
	Source     string                         `json:"source"`
	Cash       int                            `json:"cash"`
	Trading    tradingstrategy.Parameters     `json:"trading_params"`
	Indicators backtestconfig.IndicatorConfig `json:"indicators"`
}

func (actor *Actor) run(ctx context.Context, job *jobstore.Job) (downloadURL string, err error) {
	params, err := parseParameters(job.Parameters)
	if err != nil {
		err = fmt.Errorf("parsing backtest parameters: %w", err)
		return
	}
	cfg := buildConfig(params)
	replayInput := cfg.ReplayInput()
	strategy, err := replayInput.SelectStrategy()
	if err != nil {
		err = fmt.Errorf("selecting data strategy: %w", err)
		return
	}
	loaded, err := strategy.Load(ctx, replayInput)
	if err != nil {
		err = fmt.Errorf("loading price data: %w", err)
		return
	}
	result := backtest.Run(cfg, loaded.Prices, loaded.IndicatorPrices, loaded.Events)
	outputDir := fmt.Sprintf("%s/%s", actor.outputsDirectory, job.ID)
	if err = os.MkdirAll(outputDir, 0o755); err != nil {
		err = fmt.Errorf("creating output directory: %w", err)
		return
	}
	if err = writeOutputs(cfg, result, loaded, outputDir); err != nil {
		return
	}
	file, err := actor.uploadReport(ctx, job.ID, outputDir)
	if err != nil {
		err = fmt.Errorf("uploading report to storage: %w", err)
		return
	}
	downloadURL = fmt.Sprintf("/storage/v1/files/%s", file.ID)
	return
}

func (actor *Actor) uploadReport(ctx context.Context, jobID string, outputDir string) (*storageservice.File, error) {
	token, err := actor.serviceTokenMinter.MintToken()
	if err != nil {
		return nil, fmt.Errorf("minting service token: %w", err)
	}
	ctx = contextx.WithAccessToken(ctx, token)
	htmlPath := fmt.Sprintf("%s/report.html", outputDir)
	htmlData, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("reading report HTML: %w", err)
	}
	filename := fmt.Sprintf("report-%s.html", jobID)
	return storageservice.UploadFile(ctx, actor.storageClient, storageservice.UploadFileInput{
		Filename:    filename,
		ContentType: "text/html; charset=utf-8",
		Body:        bytes.NewReader(htmlData),
	})
}

func writeOutputs(cfg backtestconfig.Config, result backtest.Result, loaded *replay.LoadOutput, outputDir string) error {
	tz := tradingstrategy.USMarketLocation
	plotStart := result.Prices[0].At
	plotEnd := result.Prices[len(result.Prices)-1].At

	rsiSeries := indicator.ComputeRSI(loaded.IndicatorPrices, cfg.Indicators.RSIPeriod)
	macdSeries, macdSignalSeries := indicator.ComputeMACD(loaded.IndicatorPrices, cfg.Indicators.MACDFastPeriod, cfg.Indicators.MACDSlowPeriod, cfg.Indicators.MACDSignalPeriod)
	bollUpperSeries, bollMiddleSeries, bollLowerSeries := indicator.ComputeBollingerBands(loaded.IndicatorPrices, cfg.Indicators.BollingerPeriod, cfg.Indicators.BollingerStdDev)
	smaSeries := indicator.ComputeSMA(loaded.IndicatorPrices, cfg.Indicators.SMAPeriod)
	atrSeries := indicator.ComputeATR(loaded.IndicatorPrices, cfg.Indicators.ATRPeriod)

	tf := cfg.TradingParameters.Timeframe
	rsiPlot := filterToMarketHours(filterToRange(rsiSeries, plotStart, plotEnd), tz, tf)
	macdPlot := filterToMarketHours(filterToRange(macdSeries, plotStart, plotEnd), tz, tf)
	macdSignalPlot := filterToMarketHours(filterToRange(macdSignalSeries, plotStart, plotEnd), tz, tf)
	bollUpperPlot := filterToMarketHours(filterToRange(bollUpperSeries, plotStart, plotEnd), tz, tf)
	bollMiddlePlot := filterToMarketHours(filterToRange(bollMiddleSeries, plotStart, plotEnd), tz, tf)
	bollLowerPlot := filterToMarketHours(filterToRange(bollLowerSeries, plotStart, plotEnd), tz, tf)
	smaPlot := filterToMarketHours(filterToRange(smaSeries, plotStart, plotEnd), tz, tf)
	atrPlot := filterToMarketHours(filterToRange(atrSeries, plotStart, plotEnd), tz, tf)

	htmlPath := fmt.Sprintf("%s/report.html", outputDir)
	err := chart.RenderHTMLReport(chart.RenderHTMLReportInput{
		Symbol:       result.Symbol,
		TotalReturn:  result.TotalReturn,
		StartingCash: result.StartingCash,
		EndingCash:   result.EndingCash,
		EndingValue:  result.EndingValue,
		TradeCount:   result.TradeCount,
		WinRate:      result.WinRate,
		SharpeRatio:  result.SharpeRatio,
		Prices:       toChartPrices(result.Prices),
		Decisions:    toChartDecisions(result.Decisions),
		BollUpper:    toChartIndicator(bollUpperPlot),
		BollMiddle:   toChartIndicator(bollMiddlePlot),
		BollLower:    toChartIndicator(bollLowerPlot),
		SMA:          toChartIndicator(smaPlot),
		RSI:          toChartIndicator(rsiPlot),
		MACD:         toChartIndicator(macdPlot),
		MACDSignal:   toChartIndicator(macdSignalPlot),
		ATR:          toChartIndicator(atrPlot),
		SMAPeriod:    cfg.Indicators.SMAPeriod,
		RSIPeriod:    cfg.Indicators.RSIPeriod,
		MACDFast:     cfg.Indicators.MACDFastPeriod,
		MACDSlow:     cfg.Indicators.MACDSlowPeriod,
		MACDSignalN:  cfg.Indicators.MACDSignalPeriod,
		ATRPeriod:    cfg.Indicators.ATRPeriod,
		Timezone:     tz,
		Timeframe:    tf,
	}, htmlPath)
	if err != nil {
		return fmt.Errorf("rendering HTML report: %w", err)
	}

	decisionsPath := fmt.Sprintf("%s/decisions.txt", outputDir)
	decisionsFile, err := os.Create(decisionsPath)
	if err != nil {
		return fmt.Errorf("creating decisions file: %w", err)
	}
	writer := bufio.NewWriter(decisionsFile)
	for _, decision := range result.Decisions {
		fmt.Fprintf(writer, "%s  %-4s  price=%.4f  qty=%.4f  reason=%s\n",
			decision.At.In(tradingstrategy.USMarketLocation).Format("2006-01-02 15:04:05 MST"),
			decision.Action,
			decision.Price,
			decision.Quantity,
			decision.Reason,
		)
	}
	if err = writer.Flush(); err != nil {
		return fmt.Errorf("flushing decisions file: %w", err)
	}
	return decisionsFile.Close()
}

func parseParameters(raw map[string]string) (BacktestParameters, error) {
	var params BacktestParameters
	encoded, ok := raw["json"]
	if !ok {
		return params, fmt.Errorf("parameters missing required key 'json'")
	}
	if err := json.Unmarshal([]byte(encoded), &params); err != nil {
		return params, fmt.Errorf("decoding backtest parameters: %w", err)
	}
	if params.Symbol == "" {
		return params, fmt.Errorf("symbol is required")
	}
	if params.Start == "" {
		return params, fmt.Errorf("start is required")
	}
	if params.Cash <= 0 {
		params.Cash = 100000
	}
	if params.Source == "" {
		params.Source = "alpaca"
	}
	return params, nil
}

func buildConfig(params BacktestParameters) backtestconfig.Config {
	indicators := params.Indicators
	if indicators.RSIPeriod < 2 {
		indicators.RSIPeriod = 14
	}
	if indicators.MACDFastPeriod < 2 {
		indicators.MACDFastPeriod = 12
	}
	if indicators.MACDSlowPeriod <= indicators.MACDFastPeriod {
		indicators.MACDSlowPeriod = 26
	}
	if indicators.MACDSignalPeriod < 2 {
		indicators.MACDSignalPeriod = 9
	}
	if indicators.BollingerPeriod < 2 {
		indicators.BollingerPeriod = 20
	}
	if indicators.BollingerStdDev <= 0 {
		indicators.BollingerStdDev = 2.0
	}
	if indicators.SMAPeriod < 2 {
		indicators.SMAPeriod = 50
	}
	if indicators.ATRPeriod < 2 {
		indicators.ATRPeriod = 14
	}
	return backtestconfig.Config{
		Symbol:              params.Symbol,
		Cash:                params.Cash,
		Source:              params.Source,
		CacheEnabled:        false,
		IndicatorWarmupBars: 200,
		Start:               params.Start,
		End:                 params.End,
		Alpaca: backtestconfig.AlpacaConfig{
			Limit: 10000,
			Feed:  "iex",
		},
		Indicators:        indicators,
		TradingParameters: params.Trading,
	}
}

func filterToRange(points []indicator.Point, start, end time.Time) []indicator.Point {
	out := make([]indicator.Point, 0, len(points))
	for _, point := range points {
		if point.At.Before(start) || point.At.After(end) {
			continue
		}
		out = append(out, point)
	}
	return out
}

func filterToMarketHours(points []indicator.Point, tz *time.Location, timeframe string) []indicator.Point {
	if timeframe == "1d" || timeframe == "1w" {
		return points
	}
	out := make([]indicator.Point, 0, len(points))
	for _, point := range points {
		local := point.At.In(tz)
		hour, minute, _ := local.Clock()
		minutes := hour*60 + minute
		if minutes >= 9*60+30 && minutes <= 16*60 {
			out = append(out, point)
		}
	}
	return out
}

func toChartPrices(prices []replay.PricePoint) []chart.PricePoint {
	out := make([]chart.PricePoint, len(prices))
	for i, price := range prices {
		out[i] = chart.PricePoint{At: price.At, Close: price.Close}
	}
	return out
}

func toChartDecisions(decisions []backtest.DecisionPoint) []chart.DecisionMarker {
	out := make([]chart.DecisionMarker, len(decisions))
	for i, decision := range decisions {
		out[i] = chart.DecisionMarker{
			At:       decision.At,
			Price:    decision.Price,
			Quantity: decision.Quantity,
			IsBuy:    decision.Action == tradingstrategy.ActionBuy,
			Reason:   decision.Reason,
		}
	}
	return out
}

func toChartIndicator(points []indicator.Point) []chart.IndicatorPoint {
	out := make([]chart.IndicatorPoint, len(points))
	for i, point := range points {
		out[i] = chart.IndicatorPoint{At: point.At, Value: point.Value}
	}
	return out
}
