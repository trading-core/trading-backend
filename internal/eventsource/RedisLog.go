package eventsource

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLog struct {
	client  *redis.Client
	channel string
}

func NewRedisLog(client *redis.Client, channel string) *RedisLog {
	return &RedisLog{
		client:  client,
		channel: channel,
	}
}

func (log *RedisLog) Close() error {
	return nil
}

func (log *RedisLog) Channel() string {
	return log.channel
}

func (log *RedisLog) Append(data []byte) (event *Event, err error) {
	ctx := context.Background()
	sequence, err := log.client.Incr(ctx, log.sequenceKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("eventsource/redis: increment seq for %q: %w", log.channel, err)
	}
	_, err = log.client.XAdd(ctx, &redis.XAddArgs{
		Stream: log.channel,
		ID:     fmt.Sprintf("%d-0", sequence),
		Values: map[string]interface{}{
			"data": base64.StdEncoding.EncodeToString(data),
		},
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("eventsource/redis: XADD to %q: %w", log.channel, err)
	}
	event = &Event{
		LogID:    log.channel,
		Sequence: sequence,
		Data:     data,
	}
	return
}

func (log *RedisLog) sequenceKey() string {
	return log.channel + ":seq"
}

// Read returns up to limit events starting after cursor (exclusive).
// If timeoutMS > 0 and no events are immediately available it blocks using
// XREAD BLOCK until an event arrives or the timeout elapses.
func (log *RedisLog) Read(cursor int64, limit int, timeoutMS int64) (events []*Event, nextCursor int64, err error) {
	ctx := context.Background()
	startID := fmt.Sprintf("%d-0", cursor+1)
	messages, err := log.client.XRangeN(ctx, log.channel, startID, "+", int64(limit)).Result()
	if err != nil {
		return nil, cursor, fmt.Errorf("eventsource/redis: XRANGE on %q: %w", log.channel, err)
	}
	if len(messages) > 0 {
		return parseMessages(log.channel, messages, cursor)
	}
	// Nothing available yet; block if requested.
	if timeoutMS <= 0 {
		return nil, cursor, nil
	}
	lastID := fmt.Sprintf("%d-0", cursor)
	streams, err := log.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{log.channel, lastID},
		Count:   int64(limit),
		Block:   time.Duration(timeoutMS) * time.Millisecond,
	}).Result()
	if err == redis.Nil {
		// Timed out with no new events.
		return nil, cursor, Timeout
	}
	if err != nil {
		return nil, cursor, fmt.Errorf("eventsource/redis: XREAD BLOCK on %q: %w", log.channel, err)
	}
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, cursor, nil
	}
	return parseMessages(log.channel, streams[0].Messages, cursor)
}

// parseMessages converts raw Redis Stream messages into Events, advancing
// nextCursor to the highest sequence number seen.
func parseMessages(channel string, messages []redis.XMessage, cursor int64) (events []*Event, nextCursor int64, err error) {
	events = make([]*Event, 0, len(messages))
	nextCursor = cursor
	for _, messsage := range messages {
		sequence, err := seqFromID(messsage.ID)
		if err != nil {
			return nil, cursor, fmt.Errorf("eventsource/redis: invalid stream ID %q: %w", messsage.ID, err)
		}
		raw, ok := messsage.Values["data"].(string)
		if !ok {
			return nil, cursor, fmt.Errorf("eventsource/redis: missing data field in message %q", messsage.ID)
		}
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, cursor, fmt.Errorf("eventsource/redis: base64 decode for message %q: %w", messsage.ID, err)
		}
		events = append(events, &Event{
			LogID:    channel,
			Sequence: sequence,
			Data:     data,
		})
		if sequence > nextCursor {
			nextCursor = sequence
		}
	}
	return events, nextCursor, nil
}

// seqFromID extracts the integer sequence from a Redis Stream ID of the form
// "{seq}-0" that this implementation writes.
func seqFromID(id string) (int64, error) {
	if i := strings.IndexByte(id, '-'); i > 0 {
		return strconv.ParseInt(id[:i], 10, 64)
	}
	return strconv.ParseInt(id, 10, 64)
}
