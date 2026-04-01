package botstore

import "errors"

var (
	ErrBotNotFound  = errors.New("bot not found")
	ErrBotForbidden = errors.New("bot forbidden")
)
