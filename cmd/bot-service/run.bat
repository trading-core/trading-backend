#!/usr/bin/env bash
set -a
source ../../../trading-formation/backend/bot-service/.env
set +a
export ACCOUNT_SERVICE_HOST=localhost
go run .
