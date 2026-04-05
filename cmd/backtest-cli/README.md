# Tune.py

BACKTEST_SYMBOL=SPY BACKTEST_CACHE_ENABLED=true python tune.py --entry-mode pullback --trials 300
python tune.py --entry-mode pullback --symbols SPY,NVDA,QQQ --trials 1000 --output best.json