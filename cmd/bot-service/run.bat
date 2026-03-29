@echo off
cd /d "%~dp0"
set TRADING_API_SCHEME=http
set TRADING_API_HOST=localhost:9000
set TOKEN_SECRET=test-secret
go run main.go
