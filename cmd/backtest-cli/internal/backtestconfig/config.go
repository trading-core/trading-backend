package backtestconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

// Config holds all backtest CLI configuration, parsed and validated from
// environment variables.
type Config struct {
	Symbol              string
	Cash                int
	Source              string
	CacheEnabled        bool
	CacheDir            string
	Tune                bool
	FillLatencyMS       int
	BidAskSpreadPct     float64
	IndicatorWarmupBars int

	// Shared data-source fields — used by whichever source is active.
	Start string // RFC 3339 start time (inclusive)
	End   string // RFC 3339 end time (inclusive), may be empty

	Alpaca            AlpacaConfig
	TastyTrade        TastyTradeConfig
	Indicators        IndicatorConfig
	TradingParameters tradingstrategy.Parameters
}

type AlpacaConfig struct {
	Limit int
	Feed  string
}

type TastyTradeConfig struct {
	BrokerType        string
	CollectionTimeout time.Duration
	MaxCandles        int
}

type IndicatorConfig struct {
	RSIPeriod        int
	MACDFastPeriod   int
	MACDSlowPeriod   int
	MACDSignalPeriod int
	BollingerPeriod  int
	BollingerStdDev  float64
	SMAPeriod        int
}

// LoadFromEnv reads all backtest configuration from environment variables and
// validates constraints. Returns an error if any value is invalid.
func LoadFromEnv() Config {
	cfg := Config{
		Symbol:              config.EnvString("BACKTEST_SYMBOL", "SNDK"),
		Cash:                config.EnvInt("BACKTEST_CASH", 100000),
		Source:              config.EnvString("BACKTEST_DATA_SOURCE", "alpaca"),
		CacheEnabled:        config.EnvBool("BACKTEST_CACHE_ENABLED", false),
		CacheDir:            config.EnvString("BACKTEST_CACHE_DIR", "./tmp/cache"),
		Tune:                config.EnvBool("BACKTEST_TUNE", false),
		FillLatencyMS:       config.EnvInt("BACKTEST_FILL_LATENCY_MS", 0),
		BidAskSpreadPct:     config.EnvFloat64("BACKTEST_BID_ASK_SPREAD_PCT", 0),
		IndicatorWarmupBars: config.EnvInt("BACKTEST_INDICATOR_WARMUP_BARS", 200),

		Start: config.EnvString("BACKTEST_START", "2025-09-01T09:30:00-05:00"),
		End:   config.EnvString("BACKTEST_END", "2026-12-01T16:00:00-05:00"),

		Alpaca: AlpacaConfig{
			Limit: config.EnvInt("BACKTEST_ALPACA_STOCK_BAR_LIMIT", 10000),
			Feed:  config.EnvString("BACKTEST_ALPACA_FEED", "iex"),
		},
		TastyTrade: TastyTradeConfig{
			BrokerType:        config.EnvString("BACKTEST_TASTYTRADE_BROKER_TYPE", "tastytrade"),
			CollectionTimeout: config.EnvDuration("BACKTEST_TASTYTRADE_COLLECTION_TIMEOUT", 15*time.Second),
			MaxCandles:        config.EnvInt("BACKTEST_TASTYTRADE_MAX_CANDLES", 2500),
		},
		Indicators: IndicatorConfig{
			RSIPeriod:        config.EnvInt("BACKTEST_RSI_PERIOD", 14),
			MACDFastPeriod:   config.EnvInt("BACKTEST_MACD_FAST_PERIOD", 12),
			MACDSlowPeriod:   config.EnvInt("BACKTEST_MACD_SLOW_PERIOD", 26),
			MACDSignalPeriod: config.EnvInt("BACKTEST_MACD_SIGNAL_PERIOD", 9),
			BollingerPeriod:  config.EnvInt("BACKTEST_BOLLINGER_PERIOD", 20),
			BollingerStdDev:  config.EnvFloat64("BACKTEST_BOLLINGER_STDDEV", 2.0),
			SMAPeriod:        config.EnvInt("BACKTEST_SMA_PERIOD", 50),
		},
		// TradingParameters: tradingstrategy.Parameters{
		// 	EntryMode:                config.EnvString("BACKTEST_TRADING_PARAMETER_ENTRY_MODE", "pullback"),
		// 	MaxPositionFraction:      config.EnvFloat64("BACKTEST_TRADING_PARAMETER_MAX_POSITION_FRACTION", tradingstrategy.PullbackParameters.MaxPositionFraction),
		// 	TakeProfitPct:            config.EnvFloat64("BACKTEST_TRADING_PARAMETER_TAKE_PROFIT_PCT", tradingstrategy.PullbackParameters.TakeProfitPct),
		// 	StopLossPct:              config.EnvFloat64("BACKTEST_TRADING_PARAMETER_STOP_LOSS_PCT", tradingstrategy.PullbackParameters.StopLossPct),
		// 	SessionStart:             config.EnvInt("BACKTEST_TRADING_PARAMETER_SESSION_START", tradingstrategy.PullbackParameters.SessionStart),
		// 	SessionEnd:               config.EnvInt("BACKTEST_TRADING_PARAMETER_SESSION_END", tradingstrategy.PullbackParameters.SessionEnd),
		// 	MinRSI:                   config.EnvFloat64("BACKTEST_TRADING_PARAMETER_MIN_RSI", tradingstrategy.PullbackParameters.MinRSI),
		// 	RequireMACDSignal:        config.EnvBool("BACKTEST_TRADING_PARAMETER_REQUIRE_MACD_ABOVE_SIGNAL", true),
		// 	RequireBollingerBreakout: config.EnvBool("BACKTEST_TRADING_PARAMETER_REQUIRE_BOLLINGER_BREAKOUT", false),
		// 	MinBollingerWidthPct:     config.EnvFloat64("BACKTEST_TRADING_PARAMETER_MIN_BOLLINGER_WIDTH_PCT", 0),
		// 	MaxBollingerWidthPct:     config.EnvFloat64("BACKTEST_TRADING_PARAMETER_MAX_BOLLINGER_WIDTH_PCT", tradingstrategy.PullbackParameters.MaxBollingerWidthPct),
		// 	ReentryCooldownMinutes:   config.EnvInt("BACKTEST_TRADING_PARAMETER_REENTRY_COOLDOWN_MINUTES", tradingstrategy.PullbackParameters.ReentryCooldownMinutes),
		// 	VolatilityTPMultiplier:   config.EnvFloat64("BACKTEST_TRADING_PARAMETER_VOLATILITY_TP_MULTIPLIER", 0),
		// 	BreakoutLookbackBars:     config.EnvInt("BACKTEST_TRADING_PARAMETER_BREAKOUT_LOOKBACK_BARS", tradingstrategy.PullbackParameters.BreakoutLookbackBars),
		// 	Timeframe:                config.EnvString("BACKTEST_TRADING_PARAMETER_TIMEFRAME", "1h"),
		// },

		TradingParameters: tradingstrategy.BreakoutParameters,
	}
	if raw := os.Getenv("BACKTEST_PARAMS_JSON"); raw != "" {
		var params tradingstrategy.Parameters
		if err := json.Unmarshal([]byte(raw), &params); err != nil {
			panic(fmt.Errorf("BACKTEST_PARAMS_JSON is not valid JSON: %w", err))
		}
		cfg.TradingParameters = params
	}
	err := cfg.validate()
	if err != nil {
		panic(err)
	}
	return cfg
}

func (config Config) validate() error {
	if config.Cash < 0 {
		return fmt.Errorf("BACKTEST_CASH must be greater than zero")
	}
	if config.FillLatencyMS < 0 {
		return fmt.Errorf("BACKTEST_FILL_LATENCY_MS must be non-negative")
	}
	if config.BidAskSpreadPct < 0 {
		return fmt.Errorf("BACKTEST_BID_ASK_SPREAD_PCT must be non-negative")
	}
	if config.IndicatorWarmupBars < 0 {
		return fmt.Errorf("BACKTEST_INDICATOR_WARMUP_BARS must be non-negative")
	}

	// Indicator constraints.
	if config.Indicators.RSIPeriod < 2 {
		return fmt.Errorf("BACKTEST_RSI_PERIOD must be at least 2")
	}
	if config.Indicators.MACDFastPeriod < 2 {
		return fmt.Errorf("BACKTEST_MACD_FAST_PERIOD must be at least 2")
	}
	if config.Indicators.MACDSlowPeriod <= config.Indicators.MACDFastPeriod {
		return fmt.Errorf("BACKTEST_MACD_SLOW_PERIOD must be greater than BACKTEST_MACD_FAST_PERIOD")
	}
	if config.Indicators.MACDSignalPeriod < 2 {
		return fmt.Errorf("BACKTEST_MACD_SIGNAL_PERIOD must be at least 2")
	}
	if config.Indicators.BollingerPeriod < 2 {
		return fmt.Errorf("BACKTEST_BOLLINGER_PERIOD must be at least 2")
	}
	if config.Indicators.BollingerStdDev <= 0 {
		return fmt.Errorf("BACKTEST_BOLLINGER_STDDEV must be greater than zero")
	}
	if config.Indicators.SMAPeriod < 2 {
		return fmt.Errorf("BACKTEST_SMA_PERIOD must be at least 2")
	}

	// Trading parameters constraints.
	if config.TradingParameters.MaxPositionFraction <= 0 || config.TradingParameters.MaxPositionFraction > 1 {
		return fmt.Errorf("BACKTEST_TRADING_PARAMETERS_MAX_POSITION_FRACTION must be in (0,1]")
	}
	if config.TradingParameters.SessionStart < 0 || config.TradingParameters.SessionStart > 23 {
		return fmt.Errorf("BACKTEST_TRADING_PARAMETERS_SESSION_START must be in [0,23]")
	}
	if config.TradingParameters.SessionEnd < 1 || config.TradingParameters.SessionEnd > 24 {
		return fmt.Errorf("BACKTEST_TRADING_PARAMETERS_SESSION_END must be in [1,24]")
	}
	if config.TradingParameters.SessionStart >= config.TradingParameters.SessionEnd {
		return fmt.Errorf("BACKTEST_TRADING_PARAMETERS_SESSION_START must be less than BACKTEST_TRADING_PARAMETERS_SESSION_END")
	}
	if config.TradingParameters.StopLossPct < 0 {
		return fmt.Errorf("BACKTEST_TRADING_PARAMETERS_STOP_LOSS_PCT must be non-negative")
	}
	return nil
}

// toAlpacaTimeframe converts a compact timeframe ("1h", "1d") to the format
// Alpaca's bar API expects ("1Hour", "1Day").
func toAlpacaTimeframe(tf string) string {
	switch tf {
	case "1m":
		return "1Min"
	case "5m":
		return "5Min"
	case "10m":
		return "10Min"
	case "15m":
		return "15Min"
	case "30m":
		return "30Min"
	case "1h":
		return "1Hour"
	case "2h":
		return "2Hour"
	case "4h":
		return "4Hour"
	case "1d":
		return "1Day"
	case "1w":
		return "1Week"
	default:
		return tf
	}
}

// ReplayInput builds the replay.LoadInput from config.
func (config Config) ReplayInput() replay.LoadInput {
	return replay.LoadInput{
		Source:       config.Source,
		Symbol:       config.Symbol,
		Timeframe:    toAlpacaTimeframe(config.TradingParameters.Timeframe),
		Start:        config.Start,
		End:          config.End,
		WarmupBars:   config.IndicatorWarmupBars,
		CacheEnabled: config.CacheEnabled,
		CacheDir:     config.CacheDir,
		Alpaca: replay.AlpacaInput{
			Limit: config.Alpaca.Limit,
			Feed:  config.Alpaca.Feed,
		},
		TastyTrade: replay.TastyTradeInput{
			BrokerType:        config.TastyTrade.BrokerType,
			CollectionTimeout: config.TastyTrade.CollectionTimeout,
			MaxCandles:        config.TastyTrade.MaxCandles,
		},
	}
}

// FillLatency returns the fill latency as a time.Duration.
func (config Config) FillLatency() time.Duration {
	return time.Duration(config.FillLatencyMS) * time.Millisecond
}

// StartingCash returns Cash as float64 for backtest calculations.
func (config Config) StartingCash() float64 {
	return float64(config.Cash)
}

// OutputDir returns the output directory path for backtest results.
func (config Config) OutputDir() string {
	sourceSlug := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(config.Source)), "_", "-")
	if sourceSlug == "" {
		sourceSlug = "alpaca"
	}
	return fmt.Sprintf("./tmp/%s-%s-%s", config.Symbol, sourceSlug, config.TradingParameters.Timeframe)
}
