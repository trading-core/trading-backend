package jobstore

import "errors"

var (
	ErrJobNotFound  = errors.New("job not found")
	ErrJobForbidden = errors.New("job forbidden")
)
