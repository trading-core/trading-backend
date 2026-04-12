package httpapi

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportstore"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) DownloadReport(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	reportID := vars["report_id"]
	report, err := handler.reportQueryHandler.Get(ctx, reportID)
	if err != nil {
		err = merrifyError[err]
		return
	}
	if report.Status != reportstore.ReportStatusCompleted {
		err = merry.New("report is not yet available for download").WithHTTPCode(http.StatusConflict)
		return
	}
	htmlPath := fmt.Sprintf("%s/%s/report.html", handler.outputsDir, reportID)
	if _, statErr := os.Stat(htmlPath); os.IsNotExist(statErr) {
		err = merry.New("report file not found on disk").WithHTTPCode(http.StatusNotFound)
		return
	}
	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(responseWriter, request, htmlPath)
}
