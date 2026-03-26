package eventsource_test

import (
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/redis/go-redis/v9"
)

func TestRedisLog(t *testing.T) {
	testLog(t, "redis-backed", func(channel string) (eventsource.Log, func(), error) {
		server, err := miniredis.Run()
		if err != nil {
			return nil, nil, err
		}
		client := redis.NewClient(&redis.Options{Addr: server.Addr()})
		cleanup := func() {
			client.Close()
			server.Close()
		}
		return eventsource.NewRedisLog(client, channel), cleanup, nil
	})
}
