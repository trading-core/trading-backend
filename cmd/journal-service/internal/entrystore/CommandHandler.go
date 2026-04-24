package entrystore

import "context"

type CommandHandler interface {
	UpsertEntry(ctx context.Context, entry *Entry) error
	DeleteEntry(ctx context.Context, input DeleteEntryInput) error
}

type DeleteEntryInput struct {
	Date      string
	UpdatedAt string
}
