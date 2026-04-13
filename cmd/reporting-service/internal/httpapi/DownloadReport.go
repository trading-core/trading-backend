package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportstore"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httpx"
)

func (handler *Handler) DownloadReport(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
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
	fileID := extractFileID(report.DownloadURL)
	if fileID == "" {
		err = merry.New("report has no file attached").WithHTTPCode(http.StatusNotFound)
		return
	}
	token, err := handler.serviceTokenMinter.MintToken()
	if err != nil {
		err = fmt.Errorf("minting service token: %w", err)
		return
	}
	ctx = contextx.WithAccessToken(ctx, token)
	download, err := handler.storageClient.DownloadFile(ctx, fileID)
	if err != nil {
		return
	}
	defer download.Body.Close()
	responseWriter.Header().Set("Content-Type", download.ContentType)
	if download.ContentDisposition != "" {
		responseWriter.Header().Set("Content-Disposition", download.ContentDisposition)
	}
	_, err = io.Copy(responseWriter, download.Body)
	fatal.OnErrorUnlessDone(ctx, err)
}

// extractFileID parses the file ID from a storage-service path of the form
// /storage/v1/files/<id>.
func extractFileID(downloadURL string) string {
	const prefix = "/storage/v1/files/"
	if !strings.HasPrefix(downloadURL, prefix) {
		return ""
	}
	return strings.TrimPrefix(downloadURL, prefix)
}
