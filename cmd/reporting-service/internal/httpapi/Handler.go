package httpapi

import (
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/jobstore"
	"github.com/kduong/trading-backend/cmd/storage-service/pkg/storageservice"
	"github.com/kduong/trading-backend/internal/auth"
)

type Handler struct {
	jobCommandHandler  jobstore.CommandHandler
	jobQueryHandler    jobstore.QueryHandler
	storageClient      storageservice.Client
	serviceTokenMinter *auth.ServiceTokenMinter
	enqueueJob         func(job *jobstore.Job)
}

type NewRouterInput struct {
	AuthMiddleware     *auth.Middleware
	JobCommandHandler  jobstore.CommandHandler
	JobQueryHandler    jobstore.QueryHandler
	StorageClient      storageservice.Client
	ServiceTokenMinter *auth.ServiceTokenMinter
	EnqueueJob         func(job *jobstore.Job)
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		jobCommandHandler:  input.JobCommandHandler,
		jobQueryHandler:    input.JobQueryHandler,
		storageClient:      input.StorageClient,
		serviceTokenMinter: input.ServiceTokenMinter,
		enqueueJob:         input.EnqueueJob,
	}
	router := mux.NewRouter().StrictSlash(true)
	reportV1Router := router.PathPrefix("/reports/v1").Subrouter()
	reportV1Router.Use(input.AuthMiddleware.Handle)
	reportV1Router.HandleFunc("/jobs", handler.CreateJob).Methods(http.MethodPost).Name("CreateJob")
	reportV1Router.HandleFunc("/jobs", handler.ListJobs).Methods(http.MethodGet).Name("ListJobs")
	reportV1Router.HandleFunc("/jobs/{job_id}", handler.GetJob).Methods(http.MethodGet).Name("GetJob")
	reportV1Router.HandleFunc("/jobs/{job_id}/download", handler.DownloadJob).Methods(http.MethodGet).Name("DownloadJob")
	return router
}

var merrifyError = map[error]error{
	jobstore.ErrJobNotFound:  merry.New("job not found").WithHTTPCode(http.StatusNotFound),
	jobstore.ErrJobForbidden: merry.New("forbidden").WithHTTPCode(http.StatusForbidden),
}
