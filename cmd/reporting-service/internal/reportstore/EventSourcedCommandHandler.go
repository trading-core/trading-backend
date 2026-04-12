package reportstore

import (
	"context"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/logger"
)

var _ CommandHandler = (*EventSourcedCommandHandler)(nil)

type EventSourcedCommandHandler struct {
	log          eventsource.Log
	cursor       int64
	reportByID   map[string]*Report
}

type NewEventSourcedCommandHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedCommandHandler(input NewEventSourcedCommandHandlerInput) *EventSourcedCommandHandler {
	return &EventSourcedCommandHandler{
		log:        input.Log,
		reportByID: make(map[string]*Report),
	}
}

func (store *EventSourcedCommandHandler) Enqueue(ctx context.Context, report *Report) error {
	store.catchUp(ctx)
	if _, exists := store.reportByID[report.ID]; exists {
		logger.Fatalf("report with ID %s already exists", report.ID)
	}
	fatal.Unless(report.Status == ReportStatusPending, "new report must have status pending")
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeReportEnqueued),
		ReportEnqueuedEvent: &ReportEnqueuedEvent{
			ReportID:   report.ID,
			UserID:     report.UserID,
			Kind:       report.Kind,
			Parameters: report.Parameters,
			CreatedAt:  report.CreatedAt,
		},
	})
	_, err := store.log.Append(payload)
	return err
}

func (store *EventSourcedCommandHandler) MarkStarted(ctx context.Context, reportID string, updatedAt string) (err error) {
	store.catchUp(ctx)
	if err = store.assertOwnership(ctx, reportID); err != nil {
		return
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeReportStarted),
		ReportStartedEvent: &ReportStartedEvent{
			ReportID:  reportID,
			UpdatedAt: updatedAt,
		},
	})
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedCommandHandler) MarkCompleted(ctx context.Context, reportID string, downloadURL string, updatedAt string) (err error) {
	store.catchUp(ctx)
	if err = store.assertOwnership(ctx, reportID); err != nil {
		return
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeReportCompleted),
		ReportCompletedEvent: &ReportCompletedEvent{
			ReportID:    reportID,
			DownloadURL: downloadURL,
			UpdatedAt:   updatedAt,
		},
	})
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedCommandHandler) MarkFailed(ctx context.Context, reportID string, failReason string, updatedAt string) (err error) {
	store.catchUp(ctx)
	if err = store.assertOwnership(ctx, reportID); err != nil {
		return
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeReportFailed),
		ReportFailedEvent: &ReportFailedEvent{
			ReportID:   reportID,
			FailReason: failReason,
			UpdatedAt:  updatedAt,
		},
	})
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedCommandHandler) MarkStartedSystem(ctx context.Context, reportID string, updatedAt string) (err error) {
	store.catchUp(ctx)
	if err = store.assertExists(reportID); err != nil {
		return
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeReportStarted),
		ReportStartedEvent: &ReportStartedEvent{
			ReportID:  reportID,
			UpdatedAt: updatedAt,
		},
	})
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedCommandHandler) MarkCompletedSystem(ctx context.Context, reportID string, downloadURL string, updatedAt string) (err error) {
	store.catchUp(ctx)
	if err = store.assertExists(reportID); err != nil {
		return
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeReportCompleted),
		ReportCompletedEvent: &ReportCompletedEvent{
			ReportID:    reportID,
			DownloadURL: downloadURL,
			UpdatedAt:   updatedAt,
		},
	})
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedCommandHandler) MarkFailedSystem(ctx context.Context, reportID string, failReason string, updatedAt string) (err error) {
	store.catchUp(ctx)
	if err = store.assertExists(reportID); err != nil {
		return
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeReportFailed),
		ReportFailedEvent: &ReportFailedEvent{
			ReportID:   reportID,
			FailReason: failReason,
			UpdatedAt:  updatedAt,
		},
	})
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedCommandHandler) assertOwnership(ctx context.Context, reportID string) error {
	report, ok := store.reportByID[reportID]
	if !ok {
		return ErrReportNotFound
	}
	userID := contextx.GetUserID(ctx)
	if report.UserID != userID {
		return ErrReportForbidden
	}
	return nil
}

func (store *EventSourcedCommandHandler) assertExists(reportID string) error {
	if _, ok := store.reportByID[reportID]; !ok {
		return ErrReportNotFound
	}
	return nil
}

func (store *EventSourcedCommandHandler) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedCommandHandler) apply(ctx context.Context, event *eventsource.Event) error {
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
	}
	return nil
}

func (store *EventSourcedCommandHandler) applyEnqueued(event *ReportEnqueuedEvent) error {
	store.reportByID[event.ReportID] = &Report{
		ID:         event.ReportID,
		UserID:     event.UserID,
		Kind:       event.Kind,
		Parameters: event.Parameters,
		Status:     ReportStatusPending,
		CreatedAt:  event.CreatedAt,
		UpdatedAt:  event.CreatedAt,
	}
	return nil
}

func (store *EventSourcedCommandHandler) applyStarted(event *ReportStartedEvent) error {
	report, ok := store.reportByID[event.ReportID]
	if !ok {
		return nil
	}
	report.Status = ReportStatusRunning
	report.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedCommandHandler) applyCompleted(event *ReportCompletedEvent) error {
	report, ok := store.reportByID[event.ReportID]
	if !ok {
		return nil
	}
	report.Status = ReportStatusCompleted
	report.DownloadURL = event.DownloadURL
	report.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedCommandHandler) applyFailed(event *ReportFailedEvent) error {
	report, ok := store.reportByID[event.ReportID]
	if !ok {
		return nil
	}
	report.Status = ReportStatusFailed
	report.FailReason = event.FailReason
	report.UpdatedAt = event.UpdatedAt
	return nil
}
