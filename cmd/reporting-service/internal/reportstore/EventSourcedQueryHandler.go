package reportstore

import (
	"context"
	"sort"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ QueryHandler = (*EventSourcedQueryHandler)(nil)

type EventSourcedQueryHandler struct {
	log        eventsource.Log
	cursor     int64
	reportByID map[string]*Report
}

type NewEventSourcedQueryHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedQueryHandler(input NewEventSourcedQueryHandlerInput) *EventSourcedQueryHandler {
	return &EventSourcedQueryHandler{
		log:        input.Log,
		reportByID: make(map[string]*Report),
	}
}

func (store *EventSourcedQueryHandler) Get(ctx context.Context, reportID string) (report *Report, err error) {
	store.catchUp(ctx)
	report, ok := store.reportByID[reportID]
	if !ok {
		err = ErrReportNotFound
		return
	}
	userID := contextx.GetUserID(ctx)
	if report.UserID != userID {
		err = ErrReportForbidden
		return
	}
	return
}

func (store *EventSourcedQueryHandler) GetSystem(ctx context.Context, reportID string) (report *Report, err error) {
	store.catchUp(ctx)
	report, ok := store.reportByID[reportID]
	if !ok {
		err = ErrReportNotFound
		return
	}
	return
}

func (store *EventSourcedQueryHandler) List(ctx context.Context, input ListInput) (*ListResult, error) {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	all := make([]*Report, 0)
	for _, report := range store.reportByID {
		if report.UserID != userID {
			continue
		}
		all = append(all, report)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt > all[j].CreatedAt
	})
	totalCount := len(all)
	totalPages := totalCount / input.PageSize
	if totalCount%input.PageSize != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}
	start := input.Page * input.PageSize
	if start > totalCount {
		start = totalCount
	}
	end := start + input.PageSize
	if end > totalCount {
		end = totalCount
	}
	return &ListResult{
		Reports:    all[start:end],
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func (store *EventSourcedQueryHandler) ListAll(ctx context.Context) ([]*Report, error) {
	store.catchUp(ctx)
	result := make([]*Report, 0, len(store.reportByID))
	for _, report := range store.reportByID {
		result = append(result, report)
	}
	return result, nil
}

func (store *EventSourcedQueryHandler) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedQueryHandler) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeReportEnqueued:
		return store.applyEnqueued(frame.ReportEnqueuedEvent)
	case EventTypeReportStarted:
		return store.applyStarted(frame.ReportStartedEvent)
	case EventTypeReportCompleted:
		return store.applyCompleted(frame.ReportCompletedEvent)
	case EventTypeReportFailed:
		return store.applyFailed(frame.ReportFailedEvent)
	case EventTypeReportRetried:
		return store.applyRetried(frame.ReportRetriedEvent)
	}
	return nil
}

func (store *EventSourcedQueryHandler) applyRetried(event *ReportRetriedEvent) error {
	report, ok := store.reportByID[event.ReportID]
	if !ok {
		return nil
	}
	report.RetryCount = event.RetryCount
	report.Status = ReportStatusPending
	report.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedQueryHandler) applyEnqueued(event *ReportEnqueuedEvent) error {
	store.reportByID[event.ReportID] = &Report{
		ID:         event.ReportID,
		UserID:     event.UserID,
		Name:       event.Name,
		Kind:       event.Kind,
		Parameters: event.Parameters,
		Status:     ReportStatusPending,
		CreatedAt:  event.CreatedAt,
		UpdatedAt:  event.CreatedAt,
	}
	return nil
}

func (store *EventSourcedQueryHandler) applyStarted(event *ReportStartedEvent) error {
	report, ok := store.reportByID[event.ReportID]
	if !ok {
		return nil
	}
	report.Status = ReportStatusRunning
	report.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedQueryHandler) applyCompleted(event *ReportCompletedEvent) error {
	report, ok := store.reportByID[event.ReportID]
	if !ok {
		return nil
	}
	report.Status = ReportStatusCompleted
	report.DownloadURL = event.DownloadURL
	report.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedQueryHandler) applyFailed(event *ReportFailedEvent) error {
	report, ok := store.reportByID[event.ReportID]
	if !ok {
		return nil
	}
	report.Status = ReportStatusFailed
	report.FailReason = event.FailReason
	report.UpdatedAt = event.UpdatedAt
	return nil
}
