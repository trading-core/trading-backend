package main

import (
	"net/http"

	"github.com/kduong/trading-backend/cmd/journal-service/internal/entrystore"
	"github.com/kduong/trading-backend/cmd/journal-service/internal/httpapi"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	logFactory, err := eventsource.LogFactoryFromEnv("TRADING_JOURNAL_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("trading_journal:events")
	fatal.OnError(err)
	commandHandler := entrystore.NewCommandHandlerThreadSafeDecorator(entrystore.NewCommandHandlerThreadSafeDecoratorInput{
		Decorated: entrystore.NewEventSourcedCommandHandler(entrystore.NewEventSourcedCommandHandlerInput{
			Log: log,
		}),
	})
	queryHandler := entrystore.NewQueryHandlerThreadSafeDecorator(entrystore.NewQueryHandlerThreadSafeDecoratorInput{
		Decorated: entrystore.NewEventSourcedQueryHandler(entrystore.NewEventSourcedQueryHandlerInput{
			Log: log,
		}),
	})
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AuthMiddleware:      auth.MiddlewareFromEnv(),
		EntryCommandHandler: commandHandler,
		EntryQueryHandler:   queryHandler,
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Range", "If-Range"},
		ExposedHeaders:   []string{"Set-Cookie", "Allow", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified"},
		AllowCredentials: true,
	})
	err = http.ListenAndServe(":8084", c.Handler(router))
	fatal.OnError(err)
}
