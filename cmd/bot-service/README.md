# Bot Service

This is the backend service for managing trading bots.

## Structure

- `main.go` - Service entry point with HTTP server setup and CORS configuration
- `internal/httpapi/` - HTTP API handlers and routing
  - `Handler.go` - Request handlers for bot endpoints

## Running Locally

Execute the `run.bat` script (on Windows):
```bash
cd trading-backend/cmd/bot-service
./run.bat
```

This will start the service on `localhost:8080`.

## Docker Build

From the trading-formation directory:
```bash
docker build -f backend/bot-service/Dockerfile -t bot-service:latest ../..
```

## Environment Variables

The service uses the following environment variables (loaded via `auth.MiddlewareFromEnv()`):
- `TOKEN_SECRET` - Secret key for JWT token validation
- `TRADING_API_SCHEME` - API scheme (http/https)
- `TRADING_API_HOST` - API host

## API Endpoints

All endpoints are prefixed with `/bots/v1` and require authentication via the `Authorization` header.

Endpoints expect `X-Account-ID` header to specify the account context.

### Implemented Endpoints

- `GET /bots/v1/running` - List all running bots for the authenticated account
- `POST /bots/v1/{id}/start` - Start a bot by ID
- `POST /bots/v1/{id}/stop` - Stop a bot by ID
