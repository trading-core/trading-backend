package httpapi

import (
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportstore"
	"github.com/kduong/trading-backend/cmd/storage-service/pkg/storageservice"
	"github.com/kduong/trading-backend/internal/auth"
)

type Handler struct {
	reportCommandHandler reportstore.CommandHandler
	reportQueryHandler   reportstore.QueryHandler
	storageClient        storageservice.Client
	serviceTokenMinter   *auth.ServiceTokenMinter
	outputsDir           string
	jobs                 chan<- string
}

type NewRouterInput struct {
	AuthMiddleware       *auth.Middleware
	ReportCommandHandler reportstore.CommandHandler
	ReportQueryHandler   reportstore.QueryHandler
	StorageClient        storageservice.Client
	ServiceTokenMinter   *auth.ServiceTokenMinter
	OutputsDir           string
	Jobs                 chan<- string
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		reportCommandHandler: input.ReportCommandHandler,
		reportQueryHandler:   input.ReportQueryHandler,
		storageClient:        input.StorageClient,
		serviceTokenMinter:   input.ServiceTokenMinter,
		outputsDir:           input.OutputsDir,
		jobs:                 input.Jobs,
	}
	router := mux.NewRouter().StrictSlash(true)
	reportV1Router := router.PathPrefix("/reports/v1").Subrouter()
	reportV1Router.Use(input.AuthMiddleware.Handle)
	reportV1Router.HandleFunc("/reports", handler.EnqueueReport).Methods(http.MethodPost).Name("EnqueueReport")
	reportV1Router.HandleFunc("/reports", handler.ListReports).Methods(http.MethodGet).Name("ListReports")
	reportV1Router.HandleFunc("/reports/{report_id}", handler.GetReport).Methods(http.MethodGet).Name("GetReport")
	reportV1Router.HandleFunc("/reports/{report_id}/download", handler.DownloadReport).Methods(http.MethodGet).Name("DownloadReport")
	return router
}

var merrifyError = map[error]error{
	reportstore.ErrReportNotFound:  merry.New("report not found").WithHTTPCode(http.StatusNotFound),
	reportstore.ErrReportForbidden: merry.New("forbidden").WithHTTPCode(http.StatusForbidden),
}
