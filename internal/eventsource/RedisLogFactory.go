package eventsource

import "github.com/redis/go-redis/v9"

type RedisLogFactory struct {
	client *redis.Client
}

type NewRedisLogFactoryInput struct {
	Address string
}

func NewRedisLogFactory(input NewRedisLogFactoryInput) *RedisLogFactory {
	return &RedisLogFactory{
		client: redis.NewClient(&redis.Options{
			Addr: input.Address,
		}),
	}
}

func (factory *RedisLogFactory) Create(channel string) (log Log, err error) {
	log = &RedisLog{
		client:  factory.client,
		channel: channel,
	}
	return
}
