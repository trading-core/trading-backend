package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botsync"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httpx"
)

func (handler *Handler) StreamBotEvents(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	botID := vars["bot_id"]
	_, err = handler.botStoreQueryHandler.Get(ctx, botID)
	if err != nil {
		return
	}
	channel := handler.botChannelFunc(botID)
	log, err := handler.botEventLogFactory.Create(channel)
	fatal.OnError(err)
	flusher, ok := responseWriter.(http.Flusher)
	fatal.Unless(ok, "streaming not supported")
	responseWriter.Header().Set("Content-Type", "text/event-stream")
	responseWriter.Header().Set("Cache-Control", "no-cache")
	responseWriter.Header().Set("Connection", "keep-alive")
	responseWriter.Header().Set("X-Accel-Buffering", "no")
	_, err = subscription.Live(ctx, subscription.Input{
		Log:    log,
		Cursor: 0,
		Apply: func(ctx context.Context, event *eventsource.Event) error {
			var frame botsync.EventFrame
			err := json.Unmarshal(event.Data, &frame)
			if err != nil {
				return err
			}
			if frame.Type != botsync.EventTypeBotDecisionRecorded {
				return nil
			}
			fmt.Fprint(responseWriter, "event: decision\n")
			fmt.Fprintf(responseWriter, "id: %d\n", event.Sequence)
			fmt.Fprintf(responseWriter, "data: %s\n\n", event.Data)
			flusher.Flush()
			return nil
		},
	})
	fatal.OnError(err)
}
