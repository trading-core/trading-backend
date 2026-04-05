package backtestconfig

import (
	"fmt"
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
	Strategy            string
	Cash                int
	Source              string
	CacheEnabled        bool
	CacheDir            string
	Sweep               bool
	FillLatencyMS       int
	BidAskSpreadPct     float64
	IndicatorWarmupBars int

	// Shared data-source fields — used by whichever source is active.
	Timeframe string // candle interval, e.g. "1Min", "1Hour", "1Day"
	Start     string // RFC 3339 start time (inclusive)
	End       string // RFC 3339 end time (inclusive), may be empty

	Alpaca     AlpacaConfig
	TastyTrade TastyTradeConfig
	Indicators IndicatorConfig
	Scalping   tradingstrategy.ScalpingParams
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
}

// LoadFromEnv reads all backtest configuration from environment variables and
// validates constraints. Returns an error if any value is invalid.
func LoadFromEnv() Config {
	cfg := Config{
		Symbol:              config.EnvString("BACKTEST_SYMBOL", "SNDK"),
		Strategy:            config.EnvString("BACKTEST_STRATEGY", "scalping"),
		Cash:                config.EnvInt("BACKTEST_CASH", 100000),
		Source:              config.EnvString("BACKTEST_DATA_SOURCE", "alpaca"),
		CacheEnabled:        config.EnvBool("BACKTEST_CACHE_ENABLED", false),
		CacheDir:            config.EnvString("BACKTEST_CACHE_DIR", "./tmp/cache"),
		Sweep:               config.EnvBool("BACKTEST_SWEEP", false),
		FillLatencyMS:       config.EnvInt("BACKTEST_FILL_LATENCY_MS", 0),
		BidAskSpreadPct:     config.EnvFloat64("BACKTEST_BID_ASK_SPREAD_PCT", 0),
		IndicatorWarmupBars: config.EnvInt("BACKTEST_INDICATOR_WARMUP_BARS", 200),

		Timeframe: config.EnvString("BACKTEST_TIMEFRAME", "1Hour"),
		Start:     config.EnvString("BACKTEST_START", "2025-09-01T09:30:00-05:00"),
		End:       config.EnvString("BACKTEST_END", "2026-12-01T16:00:00-05:00"),

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
		},
		Scalping: tradingstrategy.ScalpingParams{
			EntryMode:                config.EnvString("BACKTEST_SCALPING_ENTRY_MODE", ""),
			MaxPositionFraction:      config.EnvFloat64("BACKTEST_MAX_POSITION_FRACTION", 0),
			TakeProfitPct:            config.EnvFloat64("BACKTEST_TAKE_PROFIT_PCT", 0),
			StopLossPct:              config.EnvFloat64("BACKTEST_SCALPING_STOP_LOSS_PCT", 0),
			SessionStart:             config.EnvInt("BACKTEST_SESSION_START", -1),
			SessionEnd:               config.EnvInt("BACKTEST_SESSION_END", 0),
			MinRSI:                   config.EnvFloat64("BACKTEST_SCALPING_MIN_RSI", 40),
			RequireMACDSignal:        config.EnvBool("BACKTEST_SCALPING_REQUIRE_MACD_ABOVE_SIGNAL", true),
			RequireBollingerBreakout: config.EnvBool("BACKTEST_SCALPING_REQUIRE_BOLLINGER_BREAKOUT", false),
			MinBollingerWidthPct:     config.EnvFloat64("BACKTEST_SCALPING_MIN_BOLLINGER_WIDTH_PCT", 0),
			RequireBollingerSqueeze:  config.EnvBool("BACKTEST_SCALPING_REQUIRE_BOLLINGER_SQUEEZE", false),
			MaxBollingerWidthPct:     config.EnvFloat64("BACKTEST_SCALPING_MAX_BOLLINGER_WIDTH_PCT", 0),
			ReentryCooldownMinutes:   config.EnvInt("BACKTEST_SCALPING_REENTRY_COOLDOWN_MINUTES", 0),
			UseVolatilityTP:          config.EnvBool("BACKTEST_SCALPING_USE_VOLATILITY_TP", false),
			VolatilityTPMultiplier:   config.EnvFloat64("BACKTEST_SCALPING_VOLATILITY_TP_MULTIPLIER", 0),
			RiskPerTradePct:          config.EnvFloat64("BACKTEST_SCALPING_RISK_PER_TRADE_PCT", 0),
			BreakoutLookbackBars:     config.EnvInt("BACKTEST_SCALPING_BREAKOUT_LOOKBACK_BARS", 0),
		},
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

	err := tradingstrategy.ValidateType(config.Strategy)
	if err != nil {
		return err
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

	// Scalping constraints.
	if config.Scalping.MinRSI < 0 || config.Scalping.MinRSI > 100 {
		return fmt.Errorf("BACKTEST_SCALPING_MIN_RSI must be in [0,100]")
	}
	if config.Scalping.MinBollingerWidthPct < 0 {
		return fmt.Errorf("BACKTEST_SCALPING_MIN_BOLLINGER_WIDTH_PCT must be non-negative")
	}
	if config.Scalping.StopLossPct < 0 {
		return fmt.Errorf("BACKTEST_SCALPING_STOP_LOSS_PCT must be non-negative")
	}
	if config.Scalping.RiskPerTradePct < 0 {
		return fmt.Errorf("BACKTEST_SCALPING_RISK_PER_TRADE_PCT must be non-negative")
	}
	return nil
}

// ReplayInput builds the replay.LoadInput from config.
func (config Config) ReplayInput() replay.LoadInput {
	return replay.LoadInput{
		Source:       config.Source,
		Symbol:       config.Symbol,
		Timeframe:    config.Timeframe,
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
	return fmt.Sprintf("./tmp/%s-%s-%s", config.Symbol, sourceSlug, config.Timeframe)
}
