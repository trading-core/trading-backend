package main

import (
	"net/http"

	"github.com/kduong/trading-backend/cmd/stock-screener/internal/httpapi"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker/alpaca"
	"github.com/rs/cors"
)

func main() {
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AlpacaClient:   alpaca.ClientFromEnv(),
		AuthMiddleWare: auth.MiddleWareFromEnv(),
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Range", "If-Range"},
		ExposedHeaders:   []string{"Set-Cookie", "Allow", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified"},
		AllowCredentials: true,
	})
	http.ListenAndServe(":8080", c.Handler(router))
}
