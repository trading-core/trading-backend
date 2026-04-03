Add a volatility gate
Trade only when intraday volatility is high enough (for example, ATR(14) on 1-min bars above a threshold). This avoids dead chop.

Add a liquidity + spread filter
Skip entries when spread is too wide or volume is too low. Example: spread_pct <= 0.25% and 1-min volume >= X.

Replace fixed stop with volatility stop
Use stop distance based on ATR instead of just session open loss. Example: stop at entry - 1.0 * ATR.

Add trailing profit logic
Keep your TP, but also trail after price moves in your favor (for example lock at breakeven after +0.8%, then trail by 0.5*ATR).

Add max hold time
Force exit if position is open too long (for example 30-90 minutes). This prevents stale trades eating opportunity.

Add daily risk limits
Stop trading after daily drawdown (for example -1.5% equity) or after N consecutive losses.

Add regime filter
Only run strategy when broader market regime is favorable (SPY above intraday VWAP or trend filter). This cuts bad-context trades.

Add relative volume trigger
Require RVOL above threshold (for example RVOL >= 1.5) before taking breakouts.

Time-of-day tuning by bucket
Test separate behavior for open/midday/power hour. Scalping edge is rarely uniform across the day.

Improve execution realism further
Model slippage as a function of spread/volume/volatility, not a fixed value. This will prevent over-optimistic sweeps.