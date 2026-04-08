# Tune.py

BACKTEST_SYMBOL=GOSS BACKTEST_CACHE_ENABLED=true python tune.py --trials 300
python tune.py --symbols SPY,NVDA,QQQ --trials 1000 --output best.json