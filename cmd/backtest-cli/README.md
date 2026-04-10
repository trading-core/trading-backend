# backtest-cli

A command-line tool for backtesting trading strategies against historical price data. It loads OHLCV candles from Alpaca or TastyTrade, simulates buy/sell decisions through a configurable strategy pipeline, and produces charts, an HTML report, and a decisions log.

## Build

```bash
cd trading-backend
go build -o cmd/backtest-cli/backtest-cli ./cmd/backtest-cli
```

On Windows the output is `backtest-cli.exe`.

## Running a Backtest

All configuration is via environment variables. Set them in your shell or a `.env` file sourced before running.

```bash
BACKTEST_SYMBOL=SPY \
BACKTEST_DATA_SOURCE=alpaca \
BACKTEST_START="2024-01-01T09:30:00-05:00" \
BACKTEST_END="2025-01-01T16:00:00-05:00" \
ALPACA_API_KEY=... \
ALPACA_SECRET_KEY=... \
./backtest-cli
```

Output is written to `./tmp/<SYMBOL>-<source>-<timeframe>/`:

| File | Description |
|---|---|
| `backtest.png` | Price chart with buy/sell markers and Bollinger bands |
| `backtest-with-indicators.png` | Combined chart including RSI, MACD, ATR panels |
| `indicators.png` | Indicator-only chart (RSI, MACD, ATR) |
| `report.html` | Interactive HTML report with full metrics |
| `decisions.txt` | Plain-text log of every trade decision |

Terminal output prints a summary of starting/ending cash, total return, trade count, and output file paths.

## Environment Variables

### Core

| Variable | Default | Description |
|---|---|---|
| `BACKTEST_SYMBOL` | `SNDK` | Ticker symbol to backtest |
| `BACKTEST_CASH` | `100000` | Starting cash in dollars |
| `BACKTEST_DATA_SOURCE` | `alpaca` | Data source: `alpaca` or `tastytrade` |
| `BACKTEST_START` | `2025-09-01T09:30:00-05:00` | Backtest start (RFC 3339) |
| `BACKTEST_END` | `2026-12-01T16:00:00-05:00` | Backtest end (RFC 3339); empty = now |
| `BACKTEST_FILL_LATENCY_MS` | `0` | Simulated order fill latency in ms |
| `BACKTEST_BID_ASK_SPREAD_PCT` | `0` | Simulated bid-ask spread as a fraction (e.g. `0.001` = 0.1%) |
| `BACKTEST_INDICATOR_WARMUP_BARS` | `200` | Extra historical bars loaded before `BACKTEST_START` to warm up indicators |

### Caching

| Variable | Default | Description |
|---|---|---|
| `BACKTEST_CACHE_ENABLED` | `false` | Cache API responses to disk |
| `BACKTEST_CACHE_DIR` | `./tmp/cache` | Directory for cached JSON responses |

Enable caching to avoid redundant API calls when running multiple tests or tune trials on the same symbol and date range.

### Alpaca

| Variable | Default | Description |
|---|---|---|
| `ALPACA_API_KEY` | — | Alpaca API key |
| `ALPACA_SECRET_KEY` | — | Alpaca secret key |
| `BACKTEST_ALPACA_STOCK_BAR_LIMIT` | `10000` | Max bars to fetch per request |
| `BACKTEST_ALPACA_FEED` | `iex` | Market data feed: `iex` or `sip` |

### TastyTrade

| Variable | Default | Description |
|---|---|---|
| `TASTYTRADE_USERNAME` | — | TastyTrade account username |
| `TASTYTRADE_PASSWORD` | — | TastyTrade account password |
| `BACKTEST_TASTYTRADE_BROKER_TYPE` | `tastytrade` | Broker identifier |
| `BACKTEST_TASTYTRADE_COLLECTION_TIMEOUT` | `15s` | Request timeout for candle collection |
| `BACKTEST_TASTYTRADE_MAX_CANDLES` | `2500` | Max candles to fetch |

### Indicators

| Variable | Default | Description |
|---|---|---|
| `BACKTEST_RSI_PERIOD` | `14` | RSI lookback period |
| `BACKTEST_MACD_FAST_PERIOD` | `12` | MACD fast EMA period |
| `BACKTEST_MACD_SLOW_PERIOD` | `26` | MACD slow EMA period |
| `BACKTEST_MACD_SIGNAL_PERIOD` | `9` | MACD signal EMA period |
| `BACKTEST_BOLLINGER_PERIOD` | `20` | Bollinger bands SMA period |
| `BACKTEST_BOLLINGER_STDDEV` | `2.0` | Bollinger bands standard deviation multiplier |
| `BACKTEST_SMA_PERIOD` | `50` | Simple moving average period |
| `BACKTEST_ATR_PERIOD` | `14` | ATR lookback period |

### Trading Parameters

Trading parameters are loaded from `BACKTEST_PARAMS_JSON`, a JSON object matching `tradingstrategy.Parameters`. Unset fields default to zero (which disables the corresponding feature).

```bash
BACKTEST_PARAMS_JSON='{"timeframe":"1d","max_position_fraction":0.3,"oversold_rsi":30,"overbought_rsi":70,"atr_multiplier":2.0,"lookback_bars":20}'
```

| JSON field | Description |
|---|---|
| `timeframe` | Candle interval: `1m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `1d`, `1w` |
| `max_position_fraction` | Fraction of buying power per trade (e.g. `0.3` = 30%) |
| `atr_multiplier` | ATR trailing stop multiplier; `0` disables the ATR stop |
| `risk_per_trade_pct` | Fraction of buying power to risk per trade for ATR-based sizing; `0` uses `max_position_fraction` instead |
| `oversold_rsi` | RSI threshold for oversold entry (OversoldEntryStrategy); `0` disables |
| `overbought_rsi` | RSI threshold for overbought exit (OverboughtExitStrategy); `0` disables |
| `lookback_bars` | N-bar high lookback for breakout entry (BreakoutEntryStrategy); `0` or `1` disables |
| `session_start` | Hour (0–23) for session window start; `0` disables (recommended for daily) |
| `session_end` | Hour (1–24) for session window end; `0` disables |
| `reentry_cooldown_minutes` | Minutes before re-entering after a stop-loss exit |

You can load the best params found by `tune.py` directly:

```bash
BACKTEST_PARAMS_JSON=$(cat best.json) ./backtest-cli
```

## Strategy Pipeline

The strategy evaluates every bar in this priority order:

```
SystemGuard (veto on bad market data)
└── FirstMatch:
      SessionGuard            — veto outside session window (intraday only)
      OverboughtExitStrategy  — sell when RSI overbought
      ATRStopStrategy         — trailing ATR stop loss
      PositionSizingDecorator
        └── FirstMatch (entry signals, only when flat):
              BreakoutEntryStrategy — N-bar high breakout
              TrendEntryStrategy    — MACD above signal + price above SMA
              OversoldEntryStrategy — price at lower Bollinger + RSI oversold
```

## Hyperparameter Tuning with tune.py

`tune.py` runs Bayesian optimisation (via [Optuna](https://optuna.org/)) across a set of symbols to find trading parameters that generalise rather than overfit to a single ticker.

### Setup

```bash
pip install optuna
```

Pre-warm the cache for each symbol you plan to tune (avoids live API calls during tuning):

```bash
BACKTEST_SYMBOL=SPY  BACKTEST_CACHE_ENABLED=true BACKTEST_PARAMS_JSON='{"timeframe":"1d","max_position_fraction":0.3}' ./backtest-cli
BACKTEST_SYMBOL=NVDA BACKTEST_CACHE_ENABLED=true BACKTEST_PARAMS_JSON='{"timeframe":"1d","max_position_fraction":0.3}' ./backtest-cli
BACKTEST_SYMBOL=QQQ  BACKTEST_CACHE_ENABLED=true BACKTEST_PARAMS_JSON='{"timeframe":"1d","max_position_fraction":0.3}' ./backtest-cli
```

### Usage

```bash
python tune.py [--symbols SPY,NVDA,QQQ] [--trials 300] [--binary ./backtest-cli] [--jobs 1] [--output best.json]
```

| Flag | Default | Description |
|---|---|---|
| `--symbols` | `$BACKTEST_SYMBOL` | Comma-separated tickers to tune across |
| `--trials` | `300` | Number of Optuna trials |
| `--binary` | `./backtest-cli` | Path to compiled binary |
| `--jobs` | `1` | Parallel trials (requires shared Optuna storage) |
| `--output` | — | Write best params JSON to this file |

### Objective

The optimiser maximises the **minimum adjusted Sharpe** across all symbols — the weakest symbol drives the score. This forces the parameters to generalise: a set that performs well on SPY but badly on NVDA will score only as well as NVDA. Trials with fewer than 5 completed trades on any symbol score `-inf`.

The adjusted Sharpe applies a small log-scaled boost for trade volume (capped at 20 trades), penalising parameter sets that barely trade.

### Parameters Tuned

| Parameter | Range | Notes |
|---|---|---|
| `max_position_fraction` | 0.05–0.50 | Fraction of buying power per entry |
| `oversold_rsi` | 20–40 | OversoldEntryStrategy RSI threshold |
| `overbought_rsi` | 60–80 | OverboughtExitStrategy RSI threshold |
| `lookback_bars` | 0–50 | BreakoutEntryStrategy N-bar window; 0–1 disables |
| `atr_multiplier` | 0–4 | ATRStopStrategy trailing stop; 0 disables |
| `risk_per_trade_pct` | 0–0.03 | ATR-based position sizing; 0 uses `max_position_fraction` |

### Example

```bash
# Tune across three symbols for 200 trials and save the best result
python tune.py --symbols SPY,NVDA,QQQ --trials 200 --output best.json

# Run a backtest with the best params
BACKTEST_SYMBOL=SPY BACKTEST_PARAMS_JSON=$(cat best.json) ./backtest-cli
```

---

## Testing TODO

There are currently no tests in this package. The following areas should be covered:

### `internal/backtest` — Algorithm

- **Order fill logic**: verify that a buy followed by a sell correctly updates `CashBalance`, `PositionQuantity`, and `EntryPrice`; and that the bid-ask spread is deducted correctly on both sides.
- **Fill latency**: confirm that a pending order with `FillLatencyMS > 0` is not filled until the correct bar.
- **ATR trailing stop interaction**: place a buy, advance the high, then drop price below `highSinceEntry - multiplier*ATR` and assert a sell decision is emitted.
- **Breakout lookback buffer**: assert that `LookbackHighPrice` reflects only the prior N bars, not the current bar.
- **`computeTradeReturns`**: test that buy/sell pairs are correctly paired and returns are calculated; unpaired buys (open position at end) should not produce a return.
- **`computeSharpe`**: test with a uniform return series (stddev = 0 → returns 0), and a known mean/stddev to assert the ratio.
- **`computeWinRate`**: test with all wins, all losses, and a mix.
- **`warnIfInsufficientWarmup`**: test that a warning is printed to stderr when indicator bars are below the required minimum.

### `internal/indicator` — Computations

- **`ComputeRSI`**: compare output against a known RSI series for a fixed price sequence; assert warmup bars return no point until period+1 bars are available.
- **`ComputeMACD`**: verify crossover values match a reference implementation for a small synthetic price series.
- **`ComputeBollingerBands`**: assert upper = middle + 2×stddev and lower = middle - 2×stddev for a constant price series.
- **`ComputeSMA`**: assert a rolling mean for a simple arithmetic sequence.
- **`ComputeATR`**: assert ATR converges to the average true range for a series with known high/low/close values.

### `internal/backtestconfig` — Config Loading

- **Validation**: assert that invalid values (e.g. `Cash < 0`, `MACDSlowPeriod <= MACDFastPeriod`, `MaxPositionFraction > 1`) cause a panic with the expected message.
- **`OutputDir`**: assert the path format `./tmp/<symbol>-<source>-<timeframe>` for various inputs.
- **`BACKTEST_PARAMS_JSON`**: assert that valid JSON is correctly unmarshalled into `TradingParameters`, and that invalid JSON panics.
