package accountstore

import "errors"

var (
	ErrAccountNotFound            = errors.New("account not found")
	ErrAccountForbidden           = errors.New("account forbidden")
	ErrBrokerAccountAlreadyLinked = errors.New("broker account already linked")
)
