#!/usr/bin/env bash

# Dedicated 1-hour comparison runner:
# - Loads the same environment/params from init-1hour.sh
# - Runs Alpaca and TastyTrade historical backtests with identical settings
# Usage:
#   source ./init-compare-1hour.sh
#   run_compare_1hour
# or:
#   bash ./init-compare-1hour.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./init-1hour.sh
source "$SCRIPT_DIR/init-1hour.sh"

run_compare_1hour() {
	local out_dir="${1:-./tmp/compare-${BACKTEST_SYMBOL}-1hour}"

	# Keep parameters aligned and only swap source.
	export BACKTEST_TIMEFRAME="1Hour"

	compare_backtest_sources "$out_dir"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
	run_compare_1hour "$@"
fi
