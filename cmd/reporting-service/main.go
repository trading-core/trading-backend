package main

import (
	"context"
	"net/http"
	"os"

	"github.com/kduong/trading-backend/cmd/reporting-service/internal/backtestworker"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/httpapi"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/recoveryworker"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportstore"
	"github.com/kduong/trading-backend/cmd/storage-service/pkg/storageservice"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	outputsDir := config.EnvString("REPORTING_OUTPUTS_DIR", "./tmp/reports")
	fatal.OnError(os.MkdirAll(outputsDir, 0o755))
	logFactory, err := eventsource.LogFactoryFromEnv("TRADING_REPORT_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("trading_report:events")
	fatal.OnError(err)
	commandHandler := reportstore.NewCommandHandlerThreadSafeDecorator(reportstore.NewCommandHandlerThreadSafeDecoratorInput{
		Decorated: reportstore.NewEventSourcedCommandHandler(reportstore.NewEventSourcedCommandHandlerInput{
			Log: log,
		}),
	})
	queryHandler := reportstore.NewQueryHandlerThreadSafeDecorator(reportstore.NewQueryHandlerThreadSafeDecoratorInput{
		Decorated: reportstore.NewEventSourcedQueryHandler(reportstore.NewEventSourcedQueryHandlerInput{
			Log: log,
		}),
	})
	storageClient := storageservice.ClientFromEnv()
	serviceTokenMinter := auth.ServiceTokenMinterFromEnv()
	// Buffered so the recovery worker can push recovered jobs without blocking
	// on the backtest worker starting up.
	jobs := make(chan string, 64)
	// Dead letter queue: recover any reports left in-flight from a previous crash.
	recovery := recoveryworker.New(recoveryworker.WorkerInput{
		CommandHandler: commandHandler,
		QueryHandler:   queryHandler,
		Jobs:           jobs,
	})
	recovery.Recover(ctx)
	// Backtest worker: processes jobs from the channel.
	worker := backtestworker.New(backtestworker.WorkerInput{
		CommandHandler:     commandHandler,
		QueryHandler:       queryHandler,
		StorageClient:      storageClient,
		ServiceTokenMinter: serviceTokenMinter,
		Jobs:               jobs,
		OutputsDir:         outputsDir,
	})
	go worker.Run(ctx)
	// HTTP handler also pushes new enqueued report IDs onto the job channel.
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AuthMiddleware:       auth.MiddlewareFromEnv(),
		ReportCommandHandler: commandHandler,
		ReportQueryHandler:   queryHandler,
		StorageClient:        storageClient,
		ServiceTokenMinter:   serviceTokenMinter,
		OutputsDir:           outputsDir,
		Jobs:                 jobs,
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
