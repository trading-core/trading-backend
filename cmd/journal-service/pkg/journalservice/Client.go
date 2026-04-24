package journalservice

import (
	"context"
	"errors"
	"time"

	"github.com/kduong/trading-backend/internal/config"
)

var (
	ErrEntryNotFound  = errors.New("entry not found")
	ErrEntryForbidden = errors.New("entry forbidden")
	ErrServerError    = errors.New("server error")
)

type Entry struct {
	UserID            string   `json:"user_id"`
	Date              string   `json:"date"`
	Notes             string   `json:"notes,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Mood              string   `json:"mood,omitempty"`
	DisciplineScore   int      `json:"discipline_score,omitempty"`
	ScreenshotFileIDs []string `json:"screenshot_file_ids,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

type ListResult struct {
	Entries    []*Entry `json:"entries"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	TotalCount int      `json:"total_count"`
	TotalPages int      `json:"total_pages"`
}

type ListInput struct {
	From     string
	To       string
	Page     int
	PageSize int
}

type Client interface {
	GetEntry(ctx context.Context, date string) (*Entry, error)
	ListEntries(ctx context.Context, input ListInput) (*ListResult, error)
}

func ClientFromEnv() Client {
	implementation := config.EnvStringOrFatal("JOURNAL_SERVICE_CLIENT_IMPLEMENTATION")
	switch implementation {
	case "HTTP":
		return NewHTTPClient(NewHTTPClientInput{
			Timeout: config.EnvDuration("JOURNAL_SERVICE_HTTP_CLIENT_TIMEOUT", 20*time.Second),
			BaseURL: config.EnvURLOrFatal("JOURNAL_SERVICE"),
		})
	default:
		panic("invalid journal service client implementation: " + implementation)
	}
}
