package reportstore

import "errors"

var (
	ErrReportNotFound  = errors.New("report not found")
	ErrReportForbidden = errors.New("report forbidden")
)
