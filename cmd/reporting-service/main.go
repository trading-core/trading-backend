package main

import (
	"context"
	"net/http"
	"os"

	"github.com/kduong/trading-backend/cmd/reporting-service/internal/httpapi"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/jobstore"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportsync"
	"github.com/kduong/trading-backend/cmd/storage-service/pkg/storageservice"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	outputsDirectory := config.EnvString("REPORTING_OUTPUTS_DIRECTORY", "./tmp/reports")
	err := os.MkdirAll(outputsDirectory, 0o755)
	fatal.OnError(err)
	logFactory, err := eventsource.LogFactoryFromEnv("TRADING_REPORT_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("trading_report:events")
	fatal.OnError(err)
	commandHandler := jobstore.NewCommandHandlerThreadSafeDecorator(jobstore.NewCommandHandlerThreadSafeDecoratorInput{
		Decorated: jobstore.NewEventSourcedCommandHandler(jobstore.NewEventSourcedCommandHandlerInput{
			Log: log,
		}),
	})
	queryHandler := jobstore.NewQueryHandlerThreadSafeDecorator(jobstore.NewQueryHandlerThreadSafeDecoratorInput{
		Decorated: jobstore.NewEventSourcedQueryHandler(jobstore.NewEventSourcedQueryHandlerInput{
			Log: log,
		}),
	})
	storageClient := storageservice.ClientFromEnv()
	serviceTokenMinter := auth.ServiceTokenMinterFromEnv()
	actor := reportsync.NewActor(reportsync.NewActorInput{
		CommandHandler:     commandHandler,
		StorageClient:      storageClient,
		ServiceTokenMinter: serviceTokenMinter,
		OutputsDirectory:   outputsDirectory,
		Log:                log,
	})
	actor.CatchUp(ctx)
	actor.CompleteCatchup(ctx)
	go actor.Run(ctx)
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AuthMiddleware:     auth.MiddlewareFromEnv(),
		JobCommandHandler:  commandHandler,
		JobQueryHandler:    queryHandler,
		StorageClient:      storageClient,
		ServiceTokenMinter: serviceTokenMinter,
		EnqueueJob:         actor.Notify,
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Range", "If-Range"},
		ExposedHeaders:   []string{"Set-Cookie", "Allow", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified"},
		AllowCredentials: true,
	})
	err = http.ListenAndServe(":8082", c.Handler(router))
	fatal.OnError(err)
}
