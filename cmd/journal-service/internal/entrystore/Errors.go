package entrystore

import "errors"

var (
	ErrEntryNotFound  = errors.New("entry not found")
	ErrEntryForbidden = errors.New("entry forbidden")
)
