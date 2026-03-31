package botstore

import "errors"

var (
	ErrBotAlreadyExists = errors.New("bot already exists")
	ErrBotNotFound      = errors.New("bot not found")
	ErrBotForbidden     = errors.New("bot forbidden")
)
