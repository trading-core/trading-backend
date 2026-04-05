#!/usr/bin/env python3
"""
Bayesian hyperparameter tuning for trading strategy parameters using Optuna.

Usage:
    python tune.py [--entry-mode pullback|breakout] [--symbols SPY,NVDA,QQQ] [--trials 300]

When multiple symbols are given the objective is the minimum adjusted Sharpe across all
symbols, so the optimiser is forced to find parameters that generalise rather than
overfitting to one ticker.

The backtest-cli binary must be compiled and data cached before tuning:
    BACKTEST_SYMBOL=SPY BACKTEST_CACHE_ENABLED=true ./backtest-cli   # pre-warm cache

Requirements:
    pip install optuna

Environment variables (inherited from shell):
    BACKTEST_START, BACKTEST_END, BACKTEST_TIMEFRAME, BACKTEST_DATA_SOURCE,
    BACKTEST_CACHE_DIR, ALPACA_* credentials, etc.
"""

import argparse
import json
import math
import os
import subprocess
import sys

import optuna

optuna.logging.set_verbosity(optuna.logging.WARNING)

MIN_TRADES = 5  # per-symbol minimum; fewer completed trades → -inf for that symbol


def suggest_params(trial: optuna.Trial, entry_mode: str) -> dict:
    """Build a full Parameters struct for one trial."""
    indicator_weight = trial.suggest_float("indicator_weight", 0.1, 0.9)

    # RSI range differs meaningfully between modes, so suggest it here per-mode.
    # The range is consistent within a study because entry_mode is fixed per study.
    if entry_mode == "breakout":
        min_rsi = trial.suggest_float("min_rsi", 40.0, 75.0)
    else:
        min_rsi = trial.suggest_float("min_rsi", 25.0, 65.0)

    params = {
        "entry_mode": entry_mode,
        "max_position_fraction": trial.suggest_float("max_position_fraction", 0.05, 0.50),
        "take_profit_pct": trial.suggest_float("take_profit_pct", 0.005, 0.10),
        "stop_loss_pct": trial.suggest_float("stop_loss_pct", 0.003, 0.05),
        "session_start": trial.suggest_int("session_start", 9, 12),
        "session_end": trial.suggest_int("session_end", 13, 16),
        "min_rsi": min_rsi,
        "require_macd_signal": trial.suggest_categorical("require_macd_signal", [True, False]),
        "reentry_cooldown_minutes": trial.suggest_int("reentry_cooldown_minutes", 0, 90),
        "use_volatility_tp": trial.suggest_categorical("use_volatility_tp", [True, False]),
        "volatility_tp_multiplier": trial.suggest_float("volatility_tp_multiplier", 0.3, 2.0),
        "score_buy_threshold": trial.suggest_float("score_buy_threshold", 0.40, 0.80),
        "indicator_weight": indicator_weight,
        "entry_signal_weight": round(1.0 - indicator_weight, 6),
        "risk_per_trade_pct": 0.0,
        "require_price_above_sma": trial.suggest_categorical("require_price_above_sma", [True, False]),
        "min_bollinger_width_pct": trial.suggest_float("min_bollinger_width_pct", 0.002, 0.03),
        "max_bollinger_width_pct": trial.suggest_float("max_bollinger_width_pct", 0.01, 0.05),
        "require_bollinger_squeeze": trial.suggest_categorical("require_bollinger_squeeze", [True, False]),
    }

    if entry_mode == "breakout":
        params["require_bollinger_breakout"] = trial.suggest_categorical(
            "require_bollinger_breakout", [True, False]
        )
        params["breakout_lookback_bars"] = trial.suggest_int("breakout_lookback_bars", 1, 10)
    else:
        params["require_bollinger_breakout"] = False
        params["breakout_lookback_bars"] = 1

    return params


def run_backtest(binary: str, symbol: str, params: dict, base_env: dict) -> dict | None:
    """Run the backtest-cli binary for one symbol. Returns parsed JSON or None on error."""
    env = {
        **base_env,
        "BACKTEST_SYMBOL": symbol,
        "BACKTEST_TUNE": "true",
        "BACKTEST_PARAMS_JSON": json.dumps(params),
    }
    try:
        proc = subprocess.run(
            [binary],
            env=env,
            capture_output=True,
            text=True,
            timeout=60,
        )
        if proc.returncode != 0:
            return None
        return json.loads(proc.stdout.strip())
    except (subprocess.TimeoutExpired, json.JSONDecodeError, OSError):
        return None


def adjusted_sharpe(sharpe: float, trades: int) -> float:
    """Sharpe with a small log-scaled boost for trade volume, capped at 20 trades."""
    volume_factor = math.log1p(min(trades, 20)) / math.log1p(20)
    return sharpe * (0.8 + 0.2 * volume_factor)


def objective(
    trial: optuna.Trial,
    binary: str,
    entry_mode: str,
    symbols: list[str],
    base_env: dict,
) -> float:
    params = suggest_params(trial, entry_mode)

    per_symbol: dict[str, dict] = {}
    scores: list[float] = []

    for symbol in symbols:
        metrics = run_backtest(binary, symbol, params, base_env)
        per_symbol[symbol] = metrics or {}

        if metrics is None:
            scores.append(float("-inf"))
            continue

        trades = metrics.get("trades", 0)
        sharpe = metrics.get("sharpe", 0.0)

        if trades < MIN_TRADES:
            scores.append(float("-inf"))
        else:
            scores.append(adjusted_sharpe(sharpe, trades))

    # Store per-symbol breakdown for inspection in the summary.
    for symbol, m in per_symbol.items():
        trial.set_user_attr(f"{symbol}:sharpe", m.get("sharpe", 0.0))
        trial.set_user_attr(f"{symbol}:trades", m.get("trades", 0))
        trial.set_user_attr(f"{symbol}:return", m.get("total_return", 0.0))
        trial.set_user_attr(f"{symbol}:win_rate", m.get("win_rate", 0.0))

    # Aggregates for the summary table.
    valid_metrics = [m for m in per_symbol.values() if m.get("trades", 0) >= MIN_TRADES]
    if valid_metrics:
        trial.set_user_attr("avg_sharpe", sum(m["sharpe"] for m in valid_metrics) / len(valid_metrics))
        trial.set_user_attr("total_trades", sum(m["trades"] for m in per_symbol.values()))
    else:
        trial.set_user_attr("avg_sharpe", 0.0)
        trial.set_user_attr("total_trades", 0)

    # Bottleneck objective: the weakest symbol drives the score.
    return min(scores)


def find_binary(path: str) -> str:
    if os.path.isfile(path):
        return path
    if sys.platform == "win32" and not path.endswith(".exe") and os.path.isfile(path + ".exe"):
        return path + ".exe"
    return path


def main():
    parser = argparse.ArgumentParser(description="Bayesian tuning for trading strategy parameters")
    parser.add_argument("--entry-mode", choices=["pullback", "breakout"], default="pullback")
    parser.add_argument(
        "--symbols",
        default=None,
        help="Comma-separated symbols to tune across, e.g. SPY,NVDA,QQQ. "
             "Defaults to BACKTEST_SYMBOL env var.",
    )
    parser.add_argument("--trials", type=int, default=300, help="Number of Optuna trials")
    parser.add_argument("--binary", default="./backtest-cli", help="Path to compiled backtest-cli binary")
    parser.add_argument("--jobs", type=int, default=1, help="Parallel jobs (requires shared storage)")
    parser.add_argument("--output", default=None, help="Write best params JSON to this file")
    args = parser.parse_args()

    binary = find_binary(args.binary)

    if args.symbols:
        symbols = [s.strip().upper() for s in args.symbols.split(",") if s.strip()]
    else:
        env_sym = os.environ.get("BACKTEST_SYMBOL", "").strip()
        if not env_sym:
            print("error: provide --symbols or set BACKTEST_SYMBOL", file=sys.stderr)
            sys.exit(1)
        symbols = [env_sym]

    # Inherit the current environment; force cache on so data is fetched only once.
    base_env = {**os.environ, "BACKTEST_CACHE_ENABLED": "true"}

    print(f"entry_mode : {args.entry_mode}")
    print(f"symbols    : {', '.join(symbols)}")
    print(f"trials     : {args.trials}")
    print(f"binary     : {binary}")
    print(f"objective  : min adjusted Sharpe across all symbols (bottleneck)")
    print()

    study = optuna.create_study(
        direction="maximize",
        sampler=optuna.samplers.TPESampler(seed=42),
        pruner=optuna.pruners.NopPruner(),
    )

    study.optimize(
        lambda trial: objective(trial, binary, args.entry_mode, symbols, base_env),
        n_trials=args.trials,
        n_jobs=args.jobs,
        show_progress_bar=True,
    )

    valid = [t for t in study.trials if t.value is not None and t.value > float("-inf")]
    if not valid:
        print("\nNo valid trials. Ensure the binary path is correct and data is cached.")
        sys.exit(1)

    best = max(valid, key=lambda t: t.value)

    print("\n--- Best trial ---")
    print(f"  Objective (min adjusted Sharpe): {best.value:.4f}")
    print()
    print(f"  {'Symbol':<8}  {'Sharpe':>8}  {'Return':>8}  {'WinRate':>8}  {'Trades':>6}")
    for symbol in symbols:
        print(
            f"  {symbol:<8}  "
            f"{best.user_attrs.get(f'{symbol}:sharpe', 0):>8.4f}  "
            f"{best.user_attrs.get(f'{symbol}:return', 0):>8.2%}  "
            f"{best.user_attrs.get(f'{symbol}:win_rate', 0):>8.2%}  "
            f"{best.user_attrs.get(f'{symbol}:trades', 0):>6}"
        )
    print()
    print("  Parameters:")
    best_params = suggest_params(optuna.trial.FixedTrial(best.params), args.entry_mode)
    for k, v in sorted(best_params.items()):
        print(f"    {k}: {v}")

    if args.output:
        with open(args.output, "w") as f:
            json.dump(best_params, f, indent=2)
        print(f"\nBest params written to {args.output}")

    # Top 10 summary
    top = sorted(valid, key=lambda t: t.value, reverse=True)[:10]
    print(f"\n--- Top 10 trials ---")
    sym_cols = "  ".join(f"{s:>10}" for s in symbols)
    print(f"{'#':>3}  {'Obj':>8}  {'AvgSharpe':>9}  {'Trades':>6}  {sym_cols}")
    for i, t in enumerate(top, 1):
        sharpe_cols = "  ".join(
            f"{t.user_attrs.get(f'{s}:sharpe', 0):>10.4f}" for s in symbols
        )
        print(
            f"{i:>3}  {t.value:>8.4f}  "
            f"{t.user_attrs.get('avg_sharpe', 0):>9.4f}  "
            f"{t.user_attrs.get('total_trades', 0):>6}  "
            f"{sharpe_cols}"
        )


if __name__ == "__main__":
    main()
